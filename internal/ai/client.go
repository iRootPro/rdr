// Package ai provides AI backends for article translation and summarization.
// Supports OpenAI-compatible HTTP APIs and Claude Code CLI (subscription).
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
	ProviderOpenAI   = "openai"    // OpenAI-compatible HTTP API
	ProviderClaude   = "claude"    // Claude Code CLI (subscription)
)

// Config holds the connection parameters for AI.
type Config struct {
	Provider string // "openai" or "claude"
	Endpoint string // HTTP API URL (openai provider only)
	APIKey   string // optional API key (openai provider only)
	Model    string // model name (openai) or claude model flag
}

// Enabled returns true when a usable provider is configured.
func (c Config) Enabled() bool {
	if c.Provider == ProviderClaude {
		return true
	}
	return c.Endpoint != "" && c.Model != ""
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
	if cfg.Provider == ProviderClaude {
		return completeClaude(ctx, cfg, system, user)
	}
	return completeOpenAI(ctx, cfg, system, user)
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

// completeClaude calls the Claude Code CLI via subprocess.
// Uses the user's Claude subscription (no API tokens consumed).
func completeClaude(ctx context.Context, cfg Config, system, user string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		// Fallback to common paths.
		for _, p := range []string{
			os.ExpandEnv("$HOME/.local/bin/claude"),
			"/usr/local/bin/claude",
			"/opt/homebrew/bin/claude",
		} {
			if _, serr := os.Stat(p); serr == nil {
				claudePath = p
				err = nil
				break
			}
		}
		if err != nil {
			return "", fmt.Errorf("claude CLI not found")
		}
	}

	prompt := system + "\n\n" + user
	args := []string{"--print"}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	rlog.Logf("ai/claude", "exec: %s %v | prompt: %d chars", claudePath, args, len(prompt))

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		logAI("claude", fmt.Sprintf("error: %s | stderr: %s", err, detail))
		return "", fmt.Errorf("claude: %s", detail)
	}
	out := strings.TrimSpace(stdout.String())
	logAI("claude", fmt.Sprintf("ok, %d chars", len(out)))
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
func Summarize(ctx context.Context, cfg Config, text, lang string) (string, error) {
	system := fmt.Sprintf(
		"Summarize the following article in %s. "+
			"Write 3-5 key points as a bullet list. "+
			"Be concise. Output only the summary.", lang)
	return Complete(ctx, cfg, system, trimText(text))
}
