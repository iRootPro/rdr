# rdr Phase 1 — Step 1: Project Skeleton + Database Layer

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bootstrap the Go module, create the SQLite schema with migrations, and implement CRUD methods for feeds/articles/settings — all TDD.

**Architecture:** Plain `database/sql` + `mattn/go-sqlite3`. Migrations are a
slice of SQL strings applied in a transaction, tracked by a
`schema_migrations` table. Domain types (`Feed`, `Article`) live in
`internal/db/` as plain structs. Config package ships only `ResolveHome` at
this step — YAML loading arrives in Step 2.

**Tech Stack:** Go 1.22+, `database/sql`, `github.com/mattn/go-sqlite3`,
`testing` + `t.TempDir()` for DB isolation.

**Related design doc:** `docs/plans/2026-04-13-rdr-phase1-design.md`

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Git note:** This repo lives inside a parent git repo rooted at
`/Users/sasha/Code/github.com/iRootPro`. The `rdr/` path is usually
gitignored by the parent — stage files with `git add -f rdr/...`.

---

## Task 1: Project bootstrap (go.mod, gitignore, empty main)

**Files:**
- Create: `go.mod` (via `go mod init`)
- Create: `.gitignore`
- Create: `main.go`

**Step 1.1: Initialize Go module**

Run from `rdr/`:

```bash
go mod init github.com/iRootPro/rdr
```

Expected: creates `go.mod` with `module github.com/iRootPro/rdr` and
`go 1.22` (or newer).

**Step 1.2: Create `.gitignore`**

Create `rdr/.gitignore`:

```
# build artifacts
/rdr

# local dev state
/dev/
*.db
*.db-journal

# editor / OS
.DS_Store
.vscode/
.idea/
```

**Step 1.3: Create a minimal `main.go`**

Create `rdr/main.go`:

```go
package main

import "fmt"

func main() {
    fmt.Println("rdr: bootstrap ok")
}
```

**Step 1.4: Verify build**

Run:

```bash
go build ./...
```

Expected: exits with code 0, no output. A `rdr` binary may appear — it's
gitignored.

**Step 1.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/.gitignore rdr/main.go
git commit -m "chore(rdr): bootstrap go module and gitignore"
```

---

## Task 2: Config — ResolveHome

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 2.1: Write the failing test**

Create `rdr/internal/config/config_test.go`:

```go
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
```

**Step 2.2: Run test — expect failure**

```bash
go test ./internal/config/...
```

Expected: compile error — `undefined: ResolveHome`.

**Step 2.3: Implement `ResolveHome`**

Create `rdr/internal/config/config.go`:

```go
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
```

**Step 2.4: Run test — expect pass**

```bash
go test ./internal/config/... -v
```

Expected: both tests PASS.

**Step 2.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/config/config.go rdr/internal/config/config_test.go
git commit -m "feat(rdr): add config.ResolveHome with RDR_HOME override"
```

---

## Task 3: DB package — Open and migrations

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/migrations.go`
- Create: `internal/db/db_test.go`
- Modify: `go.mod` / `go.sum` (via `go get`)

**Step 3.1: Install sqlite driver**

```bash
go get github.com/mattn/go-sqlite3@latest
```

Expected: updates `go.mod`, creates `go.sum`. Requires CGO — already enabled
on macOS by default.

**Step 3.2: Write the failing test**

Create `rdr/internal/db/db_test.go`:

```go
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
```

Note: the test touches the private `d.sql` field — that's intentional, DB
internals are tested from inside the package.

**Step 3.3: Run test — expect failure**

```bash
go test ./internal/db/...
```

Expected: compile error — undefined `Open`, `DB`.

**Step 3.4: Create `migrations.go`**

Create `rdr/internal/db/migrations.go`:

```go
package db

// migrations is an append-only list. Index+1 is the version number.
// Never edit a migration that has been released — add a new one.
var migrations = []string{
    // 001: initial schema
    `
    CREATE TABLE feeds (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        name        TEXT NOT NULL,
        url         TEXT NOT NULL UNIQUE,
        position    INTEGER NOT NULL DEFAULT 0,
        created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE articles (
        id           INTEGER PRIMARY KEY AUTOINCREMENT,
        feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
        title        TEXT NOT NULL,
        url          TEXT NOT NULL,
        description  TEXT,
        content      TEXT,
        published_at DATETIME,
        read_at      DATETIME,
        cached_at    DATETIME,
        cached_body  TEXT,
        created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(feed_id, url)
    );

    CREATE INDEX idx_articles_feed_id      ON articles(feed_id);
    CREATE INDEX idx_articles_published_at ON articles(published_at DESC);

    CREATE TABLE settings (
        key   TEXT PRIMARY KEY,
        value TEXT NOT NULL
    );

    INSERT INTO settings (key, value) VALUES
        ('refresh_interval',      '30'),
        ('max_articles_per_feed', '50'),
        ('theme',                 'dark');
    `,
}
```

**Step 3.5: Create `db.go` with Open and migration runner**

Create `rdr/internal/db/db.go`:

```go
package db

import (
    "database/sql"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

type DB struct {
    sql *sql.DB
}

func Open(path string) (*DB, error) {
    dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", path)
    raw, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }
    if err := raw.Ping(); err != nil {
        _ = raw.Close()
        return nil, fmt.Errorf("ping sqlite: %w", err)
    }
    d := &DB{sql: raw}
    if err := d.migrate(); err != nil {
        _ = raw.Close()
        return nil, fmt.Errorf("migrate: %w", err)
    }
    return d, nil
}

func (d *DB) Close() error {
    return d.sql.Close()
}

func (d *DB) migrate() error {
    if _, err := d.sql.Exec(
        `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`,
    ); err != nil {
        return err
    }

    var current int
    if err := d.sql.QueryRow(
        `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`,
    ).Scan(&current); err != nil {
        return err
    }

    for i, script := range migrations {
        version := i + 1
        if version <= current {
            continue
        }
        tx, err := d.sql.Begin()
        if err != nil {
            return err
        }
        if _, err := tx.Exec(script); err != nil {
            _ = tx.Rollback()
            return fmt.Errorf("migration %d: %w", version, err)
        }
        if _, err := tx.Exec(
            `INSERT INTO schema_migrations (version) VALUES (?)`, version,
        ); err != nil {
            _ = tx.Rollback()
            return fmt.Errorf("record migration %d: %w", version, err)
        }
        if err := tx.Commit(); err != nil {
            return fmt.Errorf("commit migration %d: %w", version, err)
        }
    }
    return nil
}
```

**Step 3.6: Run test — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: both tests PASS. If you see a CGO error, ensure Xcode Command
Line Tools are installed (`xcode-select --install`).

**Step 3.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum \
    rdr/internal/db/db.go \
    rdr/internal/db/migrations.go \
    rdr/internal/db/db_test.go
git commit -m "feat(rdr): add db package with sqlite migrations"
```

---

## Task 4: Feeds CRUD

**Files:**
- Create: `internal/db/feeds.go`
- Create: `internal/db/feeds_test.go`

**Step 4.1: Write the failing test**

Create `rdr/internal/db/feeds_test.go`:

```go
package db

import (
    "testing"
    "time"
)

func TestUpsertFeed_InsertThenUpdateName(t *testing.T) {
    d := openTestDB(t)

    f1, err := d.UpsertFeed("Hacker News", "https://hnrss.org/frontpage")
    if err != nil {
        t.Fatalf("upsert insert: %v", err)
    }
    if f1.ID == 0 || f1.Name != "Hacker News" {
        t.Fatalf("unexpected feed: %+v", f1)
    }

    f2, err := d.UpsertFeed("HN", "https://hnrss.org/frontpage")
    if err != nil {
        t.Fatalf("upsert update: %v", err)
    }
    if f2.ID != f1.ID {
        t.Fatalf("expected same id: %d vs %d", f2.ID, f1.ID)
    }
    if f2.Name != "HN" {
        t.Fatalf("expected renamed feed, got %q", f2.Name)
    }
}

func TestListFeeds_OrdersByPositionAndCountsUnread(t *testing.T) {
    d := openTestDB(t)

    a, _ := d.UpsertFeed("A", "https://a.example/rss")
    b, _ := d.UpsertFeed("B", "https://b.example/rss")

    // Insert articles directly to control state.
    mustExec(t, d, `
        INSERT INTO articles (feed_id, title, url, published_at, read_at)
        VALUES (?, 'one',   'https://a.example/1', ?, NULL),
               (?, 'two',   'https://a.example/2', ?, NULL),
               (?, 'three', 'https://a.example/3', ?, ?)
    `, a.ID, time.Now(), a.ID, time.Now(), a.ID, time.Now(), time.Now())

    mustExec(t, d, `
        INSERT INTO articles (feed_id, title, url, published_at, read_at)
        VALUES (?, 'only', 'https://b.example/1', ?, ?)
    `, b.ID, time.Now(), time.Now())

    feeds, err := d.ListFeeds()
    if err != nil {
        t.Fatalf("ListFeeds: %v", err)
    }
    if len(feeds) != 2 {
        t.Fatalf("want 2 feeds, got %d", len(feeds))
    }
    byName := map[string]Feed{feeds[0].Name: feeds[0], feeds[1].Name: feeds[1]}
    if byName["A"].UnreadCount != 2 {
        t.Fatalf("feed A unread: got %d, want 2", byName["A"].UnreadCount)
    }
    if byName["B"].UnreadCount != 0 {
        t.Fatalf("feed B unread: got %d, want 0", byName["B"].UnreadCount)
    }
}

func TestDeleteFeed_CascadesArticles(t *testing.T) {
    d := openTestDB(t)
    f, _ := d.UpsertFeed("A", "https://a.example/rss")
    mustExec(t, d, `
        INSERT INTO articles (feed_id, title, url, published_at)
        VALUES (?, 't', 'https://a.example/1', ?)
    `, f.ID, time.Now())

    if err := d.DeleteFeed(f.ID); err != nil {
        t.Fatalf("DeleteFeed: %v", err)
    }
    var count int
    if err := d.sql.QueryRow(
        `SELECT COUNT(*) FROM articles WHERE feed_id = ?`, f.ID,
    ).Scan(&count); err != nil {
        t.Fatalf("count: %v", err)
    }
    if count != 0 {
        t.Fatalf("expected cascade delete, got %d rows", count)
    }
}

func mustExec(t *testing.T, d *DB, query string, args ...any) {
    t.Helper()
    if _, err := d.sql.Exec(query, args...); err != nil {
        t.Fatalf("exec: %v", err)
    }
}
```

**Step 4.2: Run test — expect failure**

```bash
go test ./internal/db/... -run TestUpsertFeed -v
```

Expected: compile error — undefined `Feed`, `UpsertFeed`, `ListFeeds`,
`DeleteFeed`.

**Step 4.3: Implement feeds.go**

Create `rdr/internal/db/feeds.go`:

```go
package db

import (
    "database/sql"
    "errors"
    "fmt"
    "time"
)

type Feed struct {
    ID          int64
    Name        string
    URL         string
    Position    int
    CreatedAt   time.Time
    UnreadCount int
}

func (d *DB) UpsertFeed(name, url string) (Feed, error) {
    tx, err := d.sql.Begin()
    if err != nil {
        return Feed{}, err
    }
    defer func() { _ = tx.Rollback() }()

    var nextPos int
    if err := tx.QueryRow(
        `SELECT COALESCE(MAX(position), -1) + 1 FROM feeds`,
    ).Scan(&nextPos); err != nil {
        return Feed{}, fmt.Errorf("next position: %w", err)
    }

    _, err = tx.Exec(`
        INSERT INTO feeds (name, url, position) VALUES (?, ?, ?)
        ON CONFLICT(url) DO UPDATE SET name = excluded.name
    `, name, url, nextPos)
    if err != nil {
        return Feed{}, fmt.Errorf("upsert: %w", err)
    }

    var f Feed
    row := tx.QueryRow(`
        SELECT id, name, url, position, created_at
        FROM feeds WHERE url = ?
    `, url)
    if err := row.Scan(&f.ID, &f.Name, &f.URL, &f.Position, &f.CreatedAt); err != nil {
        return Feed{}, fmt.Errorf("read back: %w", err)
    }
    if err := tx.Commit(); err != nil {
        return Feed{}, err
    }
    return f, nil
}

func (d *DB) ListFeeds() ([]Feed, error) {
    rows, err := d.sql.Query(`
        SELECT f.id, f.name, f.url, f.position, f.created_at,
               COALESCE(SUM(CASE WHEN a.read_at IS NULL THEN 1 ELSE 0 END), 0)
        FROM feeds f
        LEFT JOIN articles a ON a.feed_id = f.id
        GROUP BY f.id
        ORDER BY f.position ASC, f.id ASC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Feed
    for rows.Next() {
        var f Feed
        if err := rows.Scan(
            &f.ID, &f.Name, &f.URL, &f.Position, &f.CreatedAt, &f.UnreadCount,
        ); err != nil {
            return nil, err
        }
        out = append(out, f)
    }
    return out, rows.Err()
}

func (d *DB) GetFeedByURL(url string) (*Feed, error) {
    var f Feed
    err := d.sql.QueryRow(`
        SELECT id, name, url, position, created_at
        FROM feeds WHERE url = ?
    `, url).Scan(&f.ID, &f.Name, &f.URL, &f.Position, &f.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &f, nil
}

func (d *DB) DeleteFeed(id int64) error {
    _, err := d.sql.Exec(`DELETE FROM feeds WHERE id = ?`, id)
    return err
}
```

**Step 4.4: Run tests — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: all feeds tests PASS, earlier db tests still green.

**Step 4.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/feeds.go rdr/internal/db/feeds_test.go
git commit -m "feat(rdr): add feeds CRUD with unread counts"
```

---

## Task 5: Articles CRUD

**Files:**
- Create: `internal/db/articles.go`
- Create: `internal/db/articles_test.go`

**Step 5.1: Write the failing test**

Create `rdr/internal/db/articles_test.go`:

```go
package db

import (
    "testing"
    "time"
)

func newArticle(feedID int64, url, title string, published time.Time) Article {
    return Article{
        FeedID:      feedID,
        Title:       title,
        URL:         url,
        Description: "desc",
        Content:     "<p>body</p>",
        PublishedAt: published,
    }
}

func TestUpsertArticle_InsertThenUpdatePreservesReadAt(t *testing.T) {
    d := openTestDB(t)
    f, _ := d.UpsertFeed("A", "https://a.example/rss")
    now := time.Now().UTC().Truncate(time.Second)

    a := newArticle(f.ID, "https://a.example/1", "v1", now)
    if err := d.UpsertArticle(a); err != nil {
        t.Fatalf("insert: %v", err)
    }

    list, err := d.ListArticles(f.ID, 10)
    if err != nil {
        t.Fatalf("list: %v", err)
    }
    if len(list) != 1 || list[0].Title != "v1" {
        t.Fatalf("unexpected list: %+v", list)
    }

    if err := d.MarkRead(list[0].ID); err != nil {
        t.Fatalf("MarkRead: %v", err)
    }

    a.Title = "v2"
    if err := d.UpsertArticle(a); err != nil {
        t.Fatalf("update: %v", err)
    }

    list, _ = d.ListArticles(f.ID, 10)
    if len(list) != 1 {
        t.Fatalf("expected 1 row after update, got %d", len(list))
    }
    if list[0].Title != "v2" {
        t.Fatalf("title not updated: %q", list[0].Title)
    }
    if list[0].ReadAt == nil {
        t.Fatalf("read_at was cleared by upsert, must be preserved")
    }
}

func TestListArticles_OrdersByPublishedDesc(t *testing.T) {
    d := openTestDB(t)
    f, _ := d.UpsertFeed("A", "https://a.example/rss")
    base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

    _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/1", "older", base))
    _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/2", "newer", base.Add(time.Hour)))
    _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/3", "middle", base.Add(30*time.Minute)))

    list, err := d.ListArticles(f.ID, 10)
    if err != nil {
        t.Fatalf("list: %v", err)
    }
    gotTitles := []string{list[0].Title, list[1].Title, list[2].Title}
    want := []string{"newer", "middle", "older"}
    for i := range want {
        if gotTitles[i] != want[i] {
            t.Fatalf("order: got %v, want %v", gotTitles, want)
        }
    }
}

func TestTrimArticles_RemovesOldestReadBeyondLimit(t *testing.T) {
    d := openTestDB(t)
    f, _ := d.UpsertFeed("A", "https://a.example/rss")
    base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

    // 5 articles: 3 read (old), 2 unread (newest).
    for i := 0; i < 3; i++ {
        a := newArticle(f.ID, urlN(i), "read", base.Add(time.Duration(i)*time.Minute))
        _ = d.UpsertArticle(a)
    }
    for i := 3; i < 5; i++ {
        a := newArticle(f.ID, urlN(i), "unread", base.Add(time.Duration(i)*time.Minute))
        _ = d.UpsertArticle(a)
    }
    list, _ := d.ListArticles(f.ID, 10)
    for _, a := range list {
        if a.Title == "read" {
            _ = d.MarkRead(a.ID)
        }
    }

    // Limit 3 — two oldest read articles should be removed, unread kept.
    if err := d.TrimArticles(f.ID, 3); err != nil {
        t.Fatalf("TrimArticles: %v", err)
    }

    list, _ = d.ListArticles(f.ID, 10)
    if len(list) != 3 {
        t.Fatalf("want 3 remaining, got %d", len(list))
    }
    var unread int
    for _, a := range list {
        if a.ReadAt == nil {
            unread++
        }
    }
    if unread != 2 {
        t.Fatalf("want 2 unread remaining, got %d", unread)
    }
}

func urlN(i int) string {
    return "https://a.example/" + string(rune('0'+i))
}
```

**Step 5.2: Run test — expect failure**

```bash
go test ./internal/db/... -run TestUpsertArticle -v
```

Expected: compile error — undefined `Article`, `UpsertArticle`,
`ListArticles`, `MarkRead`, `TrimArticles`.

**Step 5.3: Implement articles.go**

Create `rdr/internal/db/articles.go`:

```go
package db

import (
    "database/sql"
    "fmt"
    "time"
)

type Article struct {
    ID          int64
    FeedID      int64
    Title       string
    URL         string
    Description string
    Content     string
    PublishedAt time.Time
    ReadAt      *time.Time
    CachedAt    *time.Time
    CachedBody  string
    CreatedAt   time.Time
}

func (d *DB) UpsertArticle(a Article) error {
    _, err := d.sql.Exec(`
        INSERT INTO articles
            (feed_id, title, url, description, content, published_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(feed_id, url) DO UPDATE SET
            title        = excluded.title,
            description  = excluded.description,
            content      = excluded.content,
            published_at = excluded.published_at
    `, a.FeedID, a.Title, a.URL, a.Description, a.Content, a.PublishedAt)
    if err != nil {
        return fmt.Errorf("upsert article: %w", err)
    }
    return nil
}

func (d *DB) ListArticles(feedID int64, limit int) ([]Article, error) {
    if limit <= 0 {
        limit = 50
    }
    rows, err := d.sql.Query(`
        SELECT id, feed_id, title, url, description, content,
               published_at, read_at, cached_at, cached_body, created_at
        FROM articles
        WHERE feed_id = ?
        ORDER BY published_at DESC, id DESC
        LIMIT ?
    `, feedID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Article
    for rows.Next() {
        var (
            a          Article
            desc, cont sql.NullString
            readAt     sql.NullTime
            cachedAt   sql.NullTime
            cachedBody sql.NullString
        )
        if err := rows.Scan(
            &a.ID, &a.FeedID, &a.Title, &a.URL, &desc, &cont,
            &a.PublishedAt, &readAt, &cachedAt, &cachedBody, &a.CreatedAt,
        ); err != nil {
            return nil, err
        }
        a.Description = desc.String
        a.Content = cont.String
        if readAt.Valid {
            t := readAt.Time
            a.ReadAt = &t
        }
        if cachedAt.Valid {
            t := cachedAt.Time
            a.CachedAt = &t
        }
        a.CachedBody = cachedBody.String
        out = append(out, a)
    }
    return out, rows.Err()
}

func (d *DB) MarkRead(articleID int64) error {
    _, err := d.sql.Exec(
        `UPDATE articles SET read_at = ? WHERE id = ? AND read_at IS NULL`,
        time.Now().UTC(), articleID,
    )
    return err
}

// TrimArticles deletes the oldest read articles for a feed so that at most
// `max` rows remain. Unread articles are always kept, even if this leaves
// the feed above the limit.
func (d *DB) TrimArticles(feedID int64, max int) error {
    if max <= 0 {
        return nil
    }
    _, err := d.sql.Exec(`
        DELETE FROM articles
        WHERE id IN (
            SELECT id FROM articles
            WHERE feed_id = ? AND read_at IS NOT NULL
            ORDER BY published_at ASC, id ASC
            LIMIT MAX(0, (
                SELECT COUNT(*) FROM articles WHERE feed_id = ?
            ) - ?)
        )
    `, feedID, feedID, max)
    return err
}
```

**Step 5.4: Run tests — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: all db tests PASS.

**Step 5.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/articles.go rdr/internal/db/articles_test.go
git commit -m "feat(rdr): add articles CRUD with trim and read tracking"
```

---

## Task 6: Settings key-value API

**Files:**
- Create: `internal/db/settings.go`
- Create: `internal/db/settings_test.go`

**Step 6.1: Write the failing test**

Create `rdr/internal/db/settings_test.go`:

```go
package db

import "testing"

func TestSettings_ReadSeededDefaults(t *testing.T) {
    d := openTestDB(t)
    v, err := d.GetSetting("refresh_interval")
    if err != nil {
        t.Fatalf("GetSetting: %v", err)
    }
    if v != "30" {
        t.Fatalf("want 30, got %q", v)
    }
}

func TestSettings_SetOverwrites(t *testing.T) {
    d := openTestDB(t)
    if err := d.SetSetting("theme", "light"); err != nil {
        t.Fatalf("SetSetting: %v", err)
    }
    v, err := d.GetSetting("theme")
    if err != nil {
        t.Fatalf("GetSetting: %v", err)
    }
    if v != "light" {
        t.Fatalf("want light, got %q", v)
    }
}

func TestSettings_GetMissingReturnsEmpty(t *testing.T) {
    d := openTestDB(t)
    v, err := d.GetSetting("nope")
    if err != nil {
        t.Fatalf("GetSetting: %v", err)
    }
    if v != "" {
        t.Fatalf("want empty, got %q", v)
    }
}
```

**Step 6.2: Run test — expect failure**

```bash
go test ./internal/db/... -run TestSettings -v
```

Expected: compile error — undefined `GetSetting`, `SetSetting`.

**Step 6.3: Implement settings.go**

Create `rdr/internal/db/settings.go`:

```go
package db

import (
    "database/sql"
    "errors"
)

func (d *DB) GetSetting(key string) (string, error) {
    var v string
    err := d.sql.QueryRow(
        `SELECT value FROM settings WHERE key = ?`, key,
    ).Scan(&v)
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil
    }
    if err != nil {
        return "", err
    }
    return v, nil
}

func (d *DB) SetSetting(key, value string) error {
    _, err := d.sql.Exec(`
        INSERT INTO settings (key, value) VALUES (?, ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value
    `, key, value)
    return err
}
```

**Step 6.4: Run tests — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: all db tests PASS.

**Step 6.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/settings.go rdr/internal/db/settings_test.go
git commit -m "feat(rdr): add settings key-value API"
```

---

## Task 7: Final verification

**Step 7.1: Full test run**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go test ./... -v
```

Expected: every test PASS. If anything fails, stop and debug — do not
proceed to Step 2 of Phase 1 with a red test.

**Step 7.2: Full build**

```bash
go build ./...
```

Expected: exits 0.

**Step 7.3: Manual smoke test**

```bash
go run .
```

Expected: prints `rdr: bootstrap ok`.

**Step 7.4: Tag the step (optional)**

No tag needed — the commit history is enough. This step is complete when
all tasks above are committed and `go test ./...` is green.

---

## What's next

Step 1 delivers a tested foundation: config package with `ResolveHome`, a
migrated SQLite database, and CRUD APIs for feeds/articles/settings. Step 2
(YAML sync) wires `config.Load` + `config.Sync` into `main.go` so real feeds
land in the database.
