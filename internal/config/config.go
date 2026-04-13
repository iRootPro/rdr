package config

import (
	"fmt"
	"os"
	"path/filepath"
)

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
