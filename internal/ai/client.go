// Package ai provides a minimal OpenAI-compatible chat completion client
// used for article translation and summarization.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds the connection parameters for an OpenAI-compatible API.
type Config struct {
	Endpoint string // e.g. "http://localhost:11434/v1"
	APIKey   string // optional, "" for local models
	Model    string // e.g. "apple-foundationmodel", "llama3"
}

// Enabled returns true when a usable endpoint is configured.
func (c Config) Enabled() bool {
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

// Complete sends a chat completion request and returns the assistant reply.
func Complete(ctx context.Context, cfg Config, system, user string) (string, error) {
	if !cfg.Enabled() {
		return "", fmt.Errorf("AI not configured")
	}

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
		return "", fmt.Errorf("AI response parse error: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("AI error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI returned no choices")
	}
	return result.Choices[0].Message.Content, nil
}

// Translate sends the text for translation to the target language.
func Translate(ctx context.Context, cfg Config, text, targetLang string) (string, error) {
	system := fmt.Sprintf(
		"You are a translator. Translate the following text to %s. "+
			"Preserve formatting, paragraphs and markdown. "+
			"Output only the translation, nothing else.", targetLang)
	return Complete(ctx, cfg, system, text)
}

// Summarize sends the text for summarization.
func Summarize(ctx context.Context, cfg Config, text, lang string) (string, error) {
	system := fmt.Sprintf(
		"Summarize the following article in %s. "+
			"Write 3-5 key points as a bullet list. "+
			"Be concise. Output only the summary.", lang)
	return Complete(ctx, cfg, system, text)
}
