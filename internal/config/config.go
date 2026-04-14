package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Feeds             []FeedEntry   `yaml:"feeds"`
	SmartFolders      []SmartFolder `yaml:"smart_folders,omitempty"`
	AfterSyncCommands []string      `yaml:"after_sync_commands,omitempty"`
}

type FeedEntry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// SmartFolder is a saved query that appears as a virtual folder in the feed
// list. Selecting it loads articles from across all feeds filtered by Query
// (using the rdr query language, see internal/ui/query.go).
type SmartFolder struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

// Load reads <home>/config.yaml. A missing file yields an empty Config
// (not an error) so first-run users get a usable zero state.
func Load(home string) (*Config, error) {
	path := filepath.Join(home, "config.yaml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func ResolveHome() (string, error) {
	if v := os.Getenv("RDR_HOME"); v != "" {
		if err := os.MkdirAll(v, 0o755); err != nil {
			return "", fmt.Errorf("create RDR_HOME: %w", err)
		}
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	dir := filepath.Join(home, ".config", "rdr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}
