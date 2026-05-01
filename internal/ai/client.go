// Package ai provides AI backends for article translation and summarization.
// Supports OpenAI-compatible HTTP APIs and local AI CLIs (Claude Code, pi, opencode).
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/iRootPro/rdr/internal/rlog"
)

// maxPromptLen limits the text sent to AI to avoid timeouts and token limits.
const maxPromptLen = 8000

func logAI(provider, msg string) {
	rlog.Log("ai/"+provider, msg)
}

// Provider selects the AI backend.
const (
	ProviderOpenAI   = "openai"   // OpenAI-compatible HTTP API
	ProviderClaude   = "claude"   // Claude Code CLI (subscription)
	ProviderPi       = "pi"       // pi CLI
	ProviderOpencode = "opencode" // opencode CLI
)

// Config holds the connection parameters for AI.
type Config struct {
	Provider string // "openai", "claude", "pi", or "opencode"
	Endpoint string // HTTP API URL (openai provider only)
	APIKey   string // optional API key (openai provider only)
	Model    string // model name (openai) or optional CLI model flag
}

// Enabled returns true when a usable provider is configured.
func (c Config) Enabled() bool {
	switch c.Provider {
	case ProviderClaude, ProviderPi, ProviderOpencode:
		return true
	default:
		return c.Endpoint != "" && c.Model != ""
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a request and returns the assistant reply.
// Routes to the appropriate backend based on cfg.Provider.
func Complete(ctx context.Context, cfg Config, system, user string) (string, error) {
	if !cfg.Enabled() {
		return "", fmt.Errorf("AI not configured")
	}
	switch cfg.Provider {
	case ProviderClaude:
		return completeCLI(ctx, buildClaudeCLI(cfg, system, user))
	case ProviderPi:
		return completeCLI(ctx, buildPiCLI(cfg, system, user))
	case ProviderOpencode:
		return completeCLI(ctx, buildOpencodeCLI(cfg, system, user))
	default:
		return completeOpenAI(ctx, cfg, system, user)
	}
}

// completeOpenAI calls an OpenAI-compatible HTTP API.
func completeOpenAI(ctx context.Context, cfg Config, system, user string) (string, error) {
	body := chatRequest{
		Model: cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := cfg.Endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result chatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		logAI("openai", "parse error: "+string(data))
		return "", fmt.Errorf("AI response parse error: %w", err)
	}
	if result.Error != nil {
		logAI("openai", "api error: "+result.Error.Message)
		return "", fmt.Errorf("AI error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		logAI("openai", "no choices returned")
		return "", fmt.Errorf("AI returned no choices")
	}
	content := result.Choices[0].Message.Content
	logAI("openai", fmt.Sprintf("ok, %d chars", len(content)))
	return content, nil
}

type cliPromptMode int

const (
	cliPromptStdin cliPromptMode = iota
	cliPromptArgument
)

type cliInvocation struct {
	Provider      string
	Command       string
	Args          []string
	Prompt        string
	PromptMode    cliPromptMode
	FallbackPaths []string
}

func (c cliInvocation) finalArgs() []string {
	args := append([]string(nil), c.Args...)
	if c.PromptMode == cliPromptArgument && c.Prompt != "" {
		args = append(args, c.Prompt)
	}
	return args
}

func buildClaudeCLI(cfg Config, system, user string) cliInvocation {
	prompt := system + "\n\n" + user
	args := []string{"--print"}
	// Only pass --model if it looks like a Claude model name.
	// Ignore leftover model names from other providers (e.g. "apple-foundationmodel").
	if cfg.Model != "" && strings.HasPrefix(cfg.Model, "claude") {
		args = append(args, "--model", cfg.Model)
	}
	return cliInvocation{
		Provider:      ProviderClaude,
		Command:       "claude",
		Args:          args,
		Prompt:        prompt,
		PromptMode:    cliPromptStdin,
		FallbackPaths: commonCLIPaths("claude"),
	}
}

func buildPiCLI(cfg Config, system, user string) cliInvocation {
	args := []string{
		"--print",
		"--no-session",
		"--no-tools",
		"--no-context-files",
		"--no-extensions",
		"--no-skills",
		"--no-prompt-templates",
		"--no-themes",
		"--system-prompt", system,
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	return cliInvocation{
		Provider:      ProviderPi,
		Command:       "pi",
		Args:          args,
		Prompt:        user,
		PromptMode:    cliPromptArgument,
		FallbackPaths: commonCLIPaths("pi"),
	}
}

func buildOpencodeCLI(cfg Config, system, user string) cliInvocation {
	prompt := system + "\n\n" +
		"Do not use tools, read files, modify files, or rely on project context. " +
		"Answer only from the text below.\n\n" + user
	args := []string{"run", "--pure", "--format", "default"}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	return cliInvocation{
		Provider:      ProviderOpencode,
		Command:       "opencode",
		Args:          args,
		Prompt:        prompt,
		PromptMode:    cliPromptArgument,
		FallbackPaths: commonCLIPaths("opencode"),
	}
}

func commonCLIPaths(name string) []string {
	return []string{
		os.ExpandEnv("$HOME/.local/bin/" + name),
		"/usr/local/bin/" + name,
		"/opt/homebrew/bin/" + name,
		os.ExpandEnv("$HOME/.opencode/bin/" + name),
	}
}

func lookupCLI(command string, fallbackPaths []string) (string, error) {
	path, err := exec.LookPath(command)
	if err == nil {
		return path, nil
	}
	for _, p := range fallbackPaths {
		if p == "" {
			continue
		}
		if _, serr := os.Stat(p); serr == nil {
			return p, nil
		}
	}
	return "", err
}

func completeCLI(ctx context.Context, inv cliInvocation) (string, error) {
	path, err := lookupCLI(inv.Command, inv.FallbackPaths)
	if err != nil {
		return "", fmt.Errorf("%s CLI not found", inv.Command)
	}

	args := inv.finalArgs()
	rlog.Logf("ai/"+inv.Provider, "exec: %s %v | prompt: %d chars", path, inv.Args, len(inv.Prompt))

	cmd := exec.CommandContext(ctx, path, args...)
	if inv.PromptMode == cliPromptStdin {
		cmd.Stdin = strings.NewReader(inv.Prompt)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			detail = ctxErr.Error()
		}
		logAI(inv.Provider, fmt.Sprintf("error: %s | stderr: %s", err, detail))
		return "", fmt.Errorf("%s: %s", inv.Provider, detail)
	}
	out := strings.TrimSpace(stdout.String())
	logAI(inv.Provider, fmt.Sprintf("ok, %d chars", len(out)))
	return out, nil
}

func trimText(text string) string {
	if len(text) > maxPromptLen {
		return text[:maxPromptLen] + "\n\n[truncated]"
	}
	return text
}

// Translate sends the text for translation to the target language.
func Translate(ctx context.Context, cfg Config, text, targetLang string) (string, error) {
	system := fmt.Sprintf(
		"You are a translator. Translate the following text to %s. "+
			"Preserve formatting, paragraphs and markdown. "+
			"Output only the translation, nothing else.", targetLang)
	return Complete(ctx, cfg, system, trimText(text))
}

// Summarize sends the text for summarization.
func Summarize(ctx context.Context, cfg Config, text string) (string, error) {
	system := "Summarize the following article in the same language as the article. " +
		"If the article uses multiple languages, use its predominant language. " +
		"Write 3-5 key points as a bullet list. " +
		"Be concise. Output only the summary."
	return Complete(ctx, cfg, system, trimText(text))
}
