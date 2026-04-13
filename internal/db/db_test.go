package db

import (
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rdr.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestOpen_RunsMigrationsAndSeedsSettings(t *testing.T) {
	d := openTestDB(t)

	var version int
	if err := d.sql.QueryRow(
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`,
	).Scan(&version); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if version < 1 {
		t.Fatalf("expected migration >= 1, got %d", version)
	}

	for _, key := range []string{"refresh_interval", "max_articles_per_feed", "theme"} {
		var v string
		err := d.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
		if err != nil {
			t.Fatalf("setting %q missing: %v", key, err)
		}
		if v == "" {
			t.Fatalf("setting %q is empty", key)
		}
	}
}

func TestOpen_IsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rdr.db")

	d1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = d1.Close()

	d2, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer d2.Close()

	var count int
	if err := d2.sql.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations`,
	).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migration row, got %d", count)
	}
}
