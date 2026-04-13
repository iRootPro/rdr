package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Feeds []FeedEntry `yaml:"feeds"`
}

type FeedEntry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

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
