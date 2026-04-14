package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFileReturnsEmptyConfig(t *testing.T) {
	home := t.TempDir()
	cfg, err := Load(home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatalf("Load returned nil config")
	}
	if len(cfg.Feeds) != 0 {
		t.Fatalf("expected empty feeds, got %d", len(cfg.Feeds))
	}
}

func TestLoad_ParsesValidYAML(t *testing.T) {
	home := t.TempDir()
	body := []byte(`feeds:
  - name: Hacker News
    url: https://hnrss.org/frontpage
  - name: Go Blog
    url: https://go.dev/blog/feed.atom
`)
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), body, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := Load(home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Feeds) != 2 {
		t.Fatalf("want 2 feeds, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].Name != "Hacker News" || cfg.Feeds[0].URL != "https://hnrss.org/frontpage" {
		t.Fatalf("feed[0] mismatch: %+v", cfg.Feeds[0])
	}
	if cfg.Feeds[1].Name != "Go Blog" {
		t.Fatalf("feed[1] mismatch: %+v", cfg.Feeds[1])
	}
}

func TestLoad_ParsesSmartFolders(t *testing.T) {
	home := t.TempDir()
	body := []byte(`feeds:
  - name: Habr
    url: https://habr.com/rss
smart_folders:
  - name: Inbox
    query: unread
  - name: Today
    query: today unread
`)
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), body, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	cfg, err := Load(home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.SmartFolders) != 2 {
		t.Fatalf("want 2 folders, got %d", len(cfg.SmartFolders))
	}
	if cfg.SmartFolders[0].Name != "Inbox" || cfg.SmartFolders[0].Query != "unread" {
		t.Fatalf("folder[0] mismatch: %+v", cfg.SmartFolders[0])
	}
	if cfg.SmartFolders[1].Query != "today unread" {
		t.Fatalf("folder[1] query mismatch: %+v", cfg.SmartFolders[1])
	}
}

func TestLoad_MalformedYAMLReturnsError(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte("feeds: [unterminated"), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	if _, err := Load(home); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}
