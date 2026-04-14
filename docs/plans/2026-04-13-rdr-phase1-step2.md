# rdr Phase 1 — Step 2: YAML Sync + main wiring

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Read `config.yaml` from `RDR_HOME`, upsert its feeds into the
SQLite database without ever deleting existing feeds, and wire the
end-to-end flow in `main.go` so `RDR_HOME=./dev go run .` produces a usable
dev database.

**Architecture:** Plain `gopkg.in/yaml.v3` for parsing. `config.Load` reads
`<home>/config.yaml` (missing file → empty `Config{}`, not an error).
`config.Sync` walks the YAML entries and calls `db.UpsertFeed` for each —
the existing upsert preserves `position` on conflict, so feeds removed from
YAML stay in DB and existing feeds keep their order. `main.go` orchestrates:
`ResolveHome → db.Open → Load → Sync → ListFeeds → print`.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3`, existing `internal/db` and
`internal/config` packages, `testing` + `t.TempDir()`.

**Related design doc:** `docs/plans/2026-04-13-rdr-phase1-design.md` (see
"Конфиг и YAML sync" + "Шаг 2" sections).

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Git note:** This repo lives inside a parent git repo rooted at
`/Users/sasha/Code/github.com/iRootPro`. The `rdr/` path is gitignored by
the parent — stage files with `git add -f rdr/...`.

---

## Task 1: `config.Load` + yaml.v3 dependency (TDD)

**Files:**
- Modify: `go.mod` / `go.sum` (via `go get`)
- Create: `internal/config/load_test.go`
- Modify: `internal/config/config.go` (append `Config`, `FeedEntry`, `Load`)

**Step 1.1: Install yaml.v3**

```bash
go get gopkg.in/yaml.v3
```

Expected: updates `go.mod` with `gopkg.in/yaml.v3 vX.Y.Z`, refreshes
`go.sum`.

**Step 1.2: Write the failing test**

Create `rdr/internal/config/load_test.go`:

```go
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

func TestLoad_MalformedYAMLReturnsError(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte("feeds: [unterminated"), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	if _, err := Load(home); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}
```

**Step 1.3: Run test — expect failure**

```bash
go test ./internal/config/... -run TestLoad -v
```

Expected: compile error — `undefined: Load`, `undefined: Config`.

**Step 1.4: Implement `Config`, `FeedEntry`, `Load`**

Append to `rdr/internal/config/config.go`:

```go
import (
	// existing imports plus:
	"errors"
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
```

Note: `os` and `path/filepath` are already imported in `config.go` from
Step 1 — only add `errors` and `gopkg.in/yaml.v3`. `fmt` is also already
imported. Use the Edit tool, not Write, to avoid duplicating imports.

**Step 1.5: Run test — expect pass**

```bash
go test ./internal/config/... -v
```

Expected: 5 tests PASS (2 existing `ResolveHome` + 3 new `Load`).

**Step 1.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum \
    rdr/internal/config/config.go \
    rdr/internal/config/load_test.go
git commit -m "feat(rdr): add config.Load with yaml.v3"
```

---

## Task 2: `config.Sync` (TDD)

**Files:**
- Create: `internal/config/sync.go`
- Create: `internal/config/sync_test.go`

**Step 2.1: Write the failing test**

Create `rdr/internal/config/sync_test.go`:

```go
package config

import (
	"path/filepath"
	"testing"

	"github.com/iRootPro/rdr/internal/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "rdr.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestSync_EmptyConfigIsNoOp(t *testing.T) {
	d := openTestDB(t)
	if err := Sync(d, &Config{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	feeds, err := d.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(feeds) != 0 {
		t.Fatalf("expected no feeds, got %d", len(feeds))
	}
}

func TestSync_InsertsFeedsInYAMLOrder(t *testing.T) {
	d := openTestDB(t)
	cfg := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
		{Name: "Lobsters", URL: "https://lobste.rs/rss"},
	}}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 3 {
		t.Fatalf("want 3 feeds, got %d", len(feeds))
	}
	wantNames := []string{"HN", "Go", "Lobsters"}
	for i, f := range feeds {
		if f.Name != wantNames[i] {
			t.Fatalf("feed[%d] name: got %q, want %q", i, f.Name, wantNames[i])
		}
		if f.Position != i {
			t.Fatalf("feed[%d] position: got %d, want %d", i, f.Position, i)
		}
	}
}

func TestSync_IsIdempotent(t *testing.T) {
	d := openTestDB(t)
	cfg := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 2 {
		t.Fatalf("want 2 feeds after double sync, got %d", len(feeds))
	}
}

func TestSync_DoesNotDeleteFeedsRemovedFromYAML(t *testing.T) {
	d := openTestDB(t)
	full := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, full); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	shrunk := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
	}}
	if err := Sync(d, shrunk); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds (no deletion), got %d", len(feeds))
	}
}

func TestSync_RenamesExistingFeedAndPreservesPosition(t *testing.T) {
	d := openTestDB(t)
	first := &Config{Feeds: []FeedEntry{
		{Name: "Hacker News", URL: "https://hnrss.org/frontpage"},
		{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, first); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	feedsBefore, _ := d.ListFeeds()
	hnPos := feedsBefore[0].Position
	hnID := feedsBefore[0].ID

	renamed := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, renamed); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feedsAfter, _ := d.ListFeeds()
	var hn db.Feed
	for _, f := range feedsAfter {
		if f.URL == "https://hnrss.org/frontpage" {
			hn = f
		}
	}
	if hn.Name != "HN" {
		t.Fatalf("rename failed: %q", hn.Name)
	}
	if hn.Position != hnPos {
		t.Fatalf("position changed: %d → %d", hnPos, hn.Position)
	}
	if hn.ID != hnID {
		t.Fatalf("id changed: %d → %d", hnID, hn.ID)
	}
}
```

**Step 2.2: Run test — expect failure**

```bash
go test ./internal/config/... -run TestSync -v
```

Expected: compile error — `undefined: Sync`.

**Step 2.3: Implement `Sync`**

Create `rdr/internal/config/sync.go`:

```go
package config

import (
	"fmt"

	"github.com/iRootPro/rdr/internal/db"
)

// Sync applies the YAML config to the database: every entry is upserted
// via db.UpsertFeed, which preserves position for existing feeds and
// assigns the next available position to new ones. Feeds present in the
// database but absent from the config are NOT removed — manual deletion
// stays the user's job until the Settings TUI ships.
func Sync(d *db.DB, cfg *Config) error {
	if cfg == nil {
		return nil
	}
	for _, e := range cfg.Feeds {
		if _, err := d.UpsertFeed(e.Name, e.URL); err != nil {
			return fmt.Errorf("sync feed %q: %w", e.URL, err)
		}
	}
	return nil
}
```

**Step 2.4: Run tests — expect pass**

```bash
go test ./internal/config/... -v
```

Expected: all config tests PASS (ResolveHome + Load + Sync = 10 tests).

**Step 2.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/config/sync.go rdr/internal/config/sync_test.go
git commit -m "feat(rdr): add config.Sync upserting feeds without deletion"
```

---

## Task 3: Example `config.yaml`

**Files:**
- Create: `config.yaml`

**Step 3.1: Create the example file**

Create `rdr/config.yaml`:

```yaml
# rdr feed list — synced into ~/.config/rdr/rdr.db on every start.
# Removing entries here does NOT delete feeds from the database.
feeds:
  - name: Hacker News
    url: https://hnrss.org/frontpage
  - name: Go Blog
    url: https://go.dev/blog/feed.atom
  - name: Lobsters
    url: https://lobste.rs/rss
```

**Step 3.2: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/config.yaml
git commit -m "chore(rdr): add example config.yaml with default feeds"
```

---

## Task 4: Wire `main.go`

**Files:**
- Modify: `main.go` (replace bootstrap stub with real flow)

**Step 4.1: Replace `main.go`**

Overwrite `rdr/main.go`:

```go
package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
)

func main() {
	home, err := config.ResolveHome()
	if err != nil {
		log.Fatalf("resolve home: %v", err)
	}

	database, err := db.Open(filepath.Join(home, "rdr.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	cfg, err := config.Load(home)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := config.Sync(database, cfg); err != nil {
		log.Fatalf("sync feeds: %v", err)
	}

	feeds, err := database.ListFeeds()
	if err != nil {
		log.Fatalf("list feeds: %v", err)
	}

	fmt.Printf("rdr: home=%s, %d feed(s)\n", home, len(feeds))
	for _, f := range feeds {
		fmt.Printf("  [%d] %s — %s (unread: %d)\n",
			f.Position, f.Name, f.URL, f.UnreadCount)
	}
}
```

**Step 4.2: Build**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go build ./...
```

Expected: exits 0.

**Step 4.3: Smoke test with empty `RDR_HOME`**

```bash
rm -rf dev
RDR_HOME=./dev go run .
```

Expected output:

```
rdr: home=./dev, 0 feed(s)
```

(No `config.yaml` was copied into `./dev`, so `Load` returns empty.)

**Step 4.4: Smoke test with example `config.yaml`**

```bash
cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected output (URLs may wrap):

```
rdr: home=./dev, 3 feed(s)
  [0] Hacker News — https://hnrss.org/frontpage (unread: 0)
  [1] Go Blog — https://go.dev/blog/feed.atom (unread: 0)
  [2] Lobsters — https://lobste.rs/rss (unread: 0)
```

**Step 4.5: Smoke test idempotency**

```bash
RDR_HOME=./dev go run .
```

Expected: same output as Step 4.4 — exactly 3 feeds, same positions, no
duplicates, no errors.

**Step 4.6: Cleanup dev artifact (optional)**

```bash
rm -rf dev
```

`dev/` is gitignored anyway.

**Step 4.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/main.go
git commit -m "feat(rdr): wire main.go with ResolveHome → Open → Load → Sync"
```

---

## Task 5: Final verification

**Step 5.1: Full test run**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go test ./... -v
```

Expected: every test PASS — 13 from Step 1 plus 8 new ones (3 Load + 5
Sync) = 21 total.

**Step 5.2: Full build**

```bash
go build ./...
```

Expected: exits 0.

**Step 5.3: One more end-to-end smoke test**

```bash
rm -rf dev
mkdir -p dev
cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
RDR_HOME=./dev go run .
rm -rf dev
```

Expected: both runs print 3 feeds in the same order. No errors.

---

## What's next

Step 2 delivers a working dev loop: edit `config.yaml`, run `go run .`,
see your feeds in the database. Step 3 (Fetcher) plugs `gofeed` in so the
articles table actually fills from the network.
