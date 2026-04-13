package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHome_UsesEnvWhenSet(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "rdr-home")
	t.Setenv("RDR_HOME", custom)

	got, err := ResolveHome()
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	if got != custom {
		t.Fatalf("got %q, want %q", got, custom)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatalf("directory not created: %v", err)
	}
}

func TestResolveHome_DefaultsToXDGConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RDR_HOME", "")
	t.Setenv("HOME", dir)

	got, err := ResolveHome()
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	want := filepath.Join(dir, ".config", "rdr")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("directory not created: %v", err)
	}
}
