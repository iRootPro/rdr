package ai

import (
	"reflect"
	"strings"
	"testing"
)

func TestConfigEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"openai empty", Config{Provider: ProviderOpenAI}, false},
		{"openai endpoint only", Config{Provider: ProviderOpenAI, Endpoint: "http://localhost:11434/v1"}, false},
		{"openai model only", Config{Provider: ProviderOpenAI, Model: "llama3"}, false},
		{"openai configured", Config{Provider: ProviderOpenAI, Endpoint: "http://localhost:11434/v1", Model: "llama3"}, true},
		{"claude no endpoint model", Config{Provider: ProviderClaude}, true},
		{"pi no endpoint model", Config{Provider: ProviderPi}, true},
		{"opencode no endpoint model", Config{Provider: ProviderOpencode}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.Enabled(); got != tt.want {
				t.Fatalf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildClaudeCLI(t *testing.T) {
	inv := buildClaudeCLI(Config{Model: "claude-sonnet-4"}, "system", "user")
	if inv.Provider != ProviderClaude || inv.Command != "claude" {
		t.Fatalf("provider/command = %q/%q", inv.Provider, inv.Command)
	}
	if inv.PromptMode != cliPromptStdin {
		t.Fatalf("PromptMode = %v, want stdin", inv.PromptMode)
	}
	if inv.Prompt != "system\n\nuser" {
		t.Fatalf("Prompt = %q", inv.Prompt)
	}
	wantArgs := []string{"--print", "--model", "claude-sonnet-4"}
	if !reflect.DeepEqual(inv.finalArgs(), wantArgs) {
		t.Fatalf("args = %#v, want %#v", inv.finalArgs(), wantArgs)
	}

	inv = buildClaudeCLI(Config{Model: "gpt-4o-mini"}, "system", "user")
	wantArgs = []string{"--print"}
	if !reflect.DeepEqual(inv.finalArgs(), wantArgs) {
		t.Fatalf("non-Claude model args = %#v, want %#v", inv.finalArgs(), wantArgs)
	}
}

func TestBuildPiCLI(t *testing.T) {
	inv := buildPiCLI(Config{Model: "google/gemini-2.5-pro"}, "system", "article")
	if inv.Provider != ProviderPi || inv.Command != "pi" {
		t.Fatalf("provider/command = %q/%q", inv.Provider, inv.Command)
	}
	if inv.PromptMode != cliPromptArgument {
		t.Fatalf("PromptMode = %v, want argument", inv.PromptMode)
	}
	wantArgs := []string{
		"--print",
		"--no-session",
		"--no-tools",
		"--no-context-files",
		"--no-extensions",
		"--no-skills",
		"--no-prompt-templates",
		"--no-themes",
		"--system-prompt", "system",
		"--model", "google/gemini-2.5-pro",
		"article",
	}
	if !reflect.DeepEqual(inv.finalArgs(), wantArgs) {
		t.Fatalf("args = %#v, want %#v", inv.finalArgs(), wantArgs)
	}
}

func TestBuildOpencodeCLI(t *testing.T) {
	inv := buildOpencodeCLI(Config{}, "system", "article")
	if inv.Provider != ProviderOpencode || inv.Command != "opencode" {
		t.Fatalf("provider/command = %q/%q", inv.Provider, inv.Command)
	}
	if inv.PromptMode != cliPromptArgument {
		t.Fatalf("PromptMode = %v, want argument", inv.PromptMode)
	}
	args := inv.finalArgs()
	wantPrefix := []string{"run", "--pure", "--format", "default"}
	if len(args) != len(wantPrefix)+1 || !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args = %#v, want prefix %#v plus prompt", args, wantPrefix)
	}
	if strings.Contains(strings.Join(args[:len(wantPrefix)], " "), "--model") {
		t.Fatalf("empty model should not add --model: %#v", args)
	}
	prompt := args[len(args)-1]
	if !strings.Contains(prompt, "system") || !strings.Contains(prompt, "article") || !strings.Contains(prompt, "Do not use tools") {
		t.Fatalf("prompt does not contain expected parts: %q", prompt)
	}

	inv = buildOpencodeCLI(Config{Model: "anthropic/claude-sonnet-4"}, "system", "article")
	wantArgs := []string{"run", "--pure", "--format", "default", "--model", "anthropic/claude-sonnet-4", inv.Prompt}
	if !reflect.DeepEqual(inv.finalArgs(), wantArgs) {
		t.Fatalf("model args = %#v, want %#v", inv.finalArgs(), wantArgs)
	}
}
