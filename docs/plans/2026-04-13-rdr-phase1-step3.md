# rdr Phase 1 — Step 3: Fetcher (gofeed + parallel HTTP)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Pull RSS/Atom feeds over HTTP, parse them with `gofeed`, upsert
articles into SQLite, trim old read articles to the per-feed cap, and run
all of it in parallel via `errgroup` with a 8-goroutine semaphore. Expose
the flow through a temporary `--fetch` flag in `main.go`.

**Architecture:** New package `internal/feed/` with a `Fetcher` that owns a
`*db.DB`, an `*http.Client` (15s timeout, custom UA), and a
`*gofeed.Parser`. `FetchOne` does HTTP GET → parse → map gofeed.Item →
db.Article → upsert loop → trim. `FetchAll` lists feeds from DB and runs
`FetchOne` per feed inside `errgroup.WithContext` + buffered semaphore;
per-feed errors land in `FetchResult.Err`, never aborting siblings. To
report Added vs Updated we extend `db.UpsertArticle` to return
`(inserted bool, err error)` (smallest possible API change to Step 1 code).

**Tech Stack:** Go 1.22+, `github.com/mmcdole/gofeed`,
`golang.org/x/sync/errgroup`, `net/http/httptest` for tests, existing
`internal/db` and `internal/config` packages.

**Related design doc:** `docs/plans/2026-04-13-rdr-phase1-design.md`
(see "Fetcher" + "Шаг 3").

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Git note:** This repo lives inside a parent git repo rooted at
`/Users/sasha/Code/github.com/iRootPro`. The `rdr/` path is gitignored by
the parent — stage with `git add -f rdr/...`.

---

## Task 1: Add gofeed + errgroup dependencies

**Files:**
- Modify: `go.mod` / `go.sum` (via `go get`)

**Step 1.1: Install both deps**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go get github.com/mmcdole/gofeed
go get golang.org/x/sync/errgroup
go mod tidy
```

Expected: `go.mod` gains direct entries for `github.com/mmcdole/gofeed` and
`golang.org/x/sync`. `go.sum` updated.

**Step 1.2: Sanity check build still works**

```bash
go build ./...
```

Expected: exits 0.

**Step 1.3: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum
git commit -m "chore(rdr): add gofeed and errgroup deps"
```

---

## Task 2: `db.UpsertArticle` returns `(inserted bool, error)`

The fetcher needs to know whether each article was new or updated, to
report Added/Updated counts. The smallest change is to widen
`UpsertArticle`'s return signature. Existing callers (3 in
`articles_test.go`) get a one-character fix.

**Files:**
- Modify: `internal/db/articles.go:23-38`
- Modify: `internal/db/articles_test.go` (3 call sites + 1 new test)

**Step 2.1: Write the failing test**

Append to `rdr/internal/db/articles_test.go`, **at the end of the file**:

```go
func TestUpsertArticle_ReturnsInsertedFlag(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	a := newArticle(f.ID, "https://a.example/1", "v1", time.Now().UTC())

	inserted, err := d.UpsertArticle(a)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if !inserted {
		t.Fatalf("first upsert: expected inserted=true")
	}

	inserted, err = d.UpsertArticle(a)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if inserted {
		t.Fatalf("second upsert: expected inserted=false")
	}
}
```

This test will not compile until we change the signature. Good — it forces
the change.

**Step 2.2: Run test — expect failure**

```bash
go test ./internal/db/... -run TestUpsertArticle_ReturnsInsertedFlag -v
```

Expected: compile error — `assignment mismatch: 2 variables but
d.UpsertArticle returns 1 value`.

**Step 2.3: Update `UpsertArticle` signature**

Edit `rdr/internal/db/articles.go` — replace the existing `UpsertArticle`
function (lines 23-38) with:

```go
func (d *DB) UpsertArticle(a Article) (bool, error) {
	var existed int
	err := d.sql.QueryRow(
		`SELECT 1 FROM articles WHERE feed_id = ? AND url = ?`,
		a.FeedID, a.URL,
	).Scan(&existed)
	inserted := errors.Is(err, sql.ErrNoRows)
	if err != nil && !inserted {
		return false, fmt.Errorf("check article: %w", err)
	}

	_, err = d.sql.Exec(`
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
		return false, fmt.Errorf("upsert article: %w", err)
	}
	return inserted, nil
}
```

The file already imports `database/sql`, `fmt`, `time`. We need `errors`
too — add it to the import block.

**Step 2.4: Fix existing call sites in `articles_test.go`**

Three lines need a one-character fix. Use `replace_all = false` and target
each:

In `TestUpsertArticle_InsertThenUpdatePreservesReadAt`:
- `if err := d.UpsertArticle(a); err != nil {` (the insert call) →
  `if _, err := d.UpsertArticle(a); err != nil {`
- `if err := d.UpsertArticle(a); err != nil {` (the update call) →
  `if _, err := d.UpsertArticle(a); err != nil {`

In `TestListArticles_OrdersByPublishedDesc`:
- `_ = d.UpsertArticle(newArticle(...))` × 3 — these already discard the
  return; widen each to `_, _ = d.UpsertArticle(newArticle(...))`.

In `TestTrimArticles_RemovesOldestReadBeyondLimit`:
- `_ = d.UpsertArticle(a)` × 2 (one in each loop) — widen each to
  `_, _ = d.UpsertArticle(a)`.

If you're unsure how many call sites there are, run:
```bash
grep -n "UpsertArticle" rdr/internal/db/articles_test.go
```
and fix every line.

**Step 2.5: Run all db tests — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: every db test PASS, including the new
`TestUpsertArticle_ReturnsInsertedFlag`.

**Step 2.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/articles.go rdr/internal/db/articles_test.go
git commit -m "refactor(rdr): UpsertArticle returns inserted bool for fetch counts"
```

---

## Task 3: Fetcher skeleton + happy-path FetchOne (TDD)

**Files:**
- Create: `internal/feed/testdata/atom_feed.xml`
- Create: `internal/feed/fetcher_test.go`
- Create: `internal/feed/fetcher.go`

**Step 3.1: Create the Atom test fixture**

Create `rdr/internal/feed/testdata/atom_feed.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Feed</title>
  <link href="http://example.org/"/>
  <updated>2026-04-13T12:00:00Z</updated>
  <id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>
  <entry>
    <title>First post</title>
    <link href="http://example.org/2026/04/13/first"/>
    <id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2026-04-13T11:00:00Z</updated>
    <published>2026-04-13T11:00:00Z</published>
    <summary>First post summary</summary>
    <content type="html">&lt;p&gt;First post body&lt;/p&gt;</content>
  </entry>
  <entry>
    <title>Second post</title>
    <link href="http://example.org/2026/04/12/second"/>
    <id>urn:uuid:2225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2026-04-12T10:00:00Z</updated>
    <published>2026-04-12T10:00:00Z</published>
    <summary>Second post summary</summary>
    <content type="html">&lt;p&gt;Second post body&lt;/p&gt;</content>
  </entry>
  <entry>
    <title>Third post</title>
    <link href="http://example.org/2026/04/11/third"/>
    <id>urn:uuid:3225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2026-04-11T09:00:00Z</updated>
    <published>2026-04-11T09:00:00Z</published>
    <summary>Third post summary</summary>
    <content type="html">&lt;p&gt;Third post body&lt;/p&gt;</content>
  </entry>
</feed>
```

(`testdata/` is automatically excluded from Go builds, so no special
gitignore work needed.)

**Step 3.2: Write the failing happy-path test**

Create `rdr/internal/feed/fetcher_test.go`:

```go
package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

func serveFixture(t *testing.T, name string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write(body)
	}))
}

func TestFetchOne_AtomHappyPath(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()

	feed, _ := d.UpsertFeed("Example", srv.URL)

	f := New(d)
	result, err := f.FetchOne(context.Background(), feed)
	if err != nil {
		t.Fatalf("FetchOne: %v", err)
	}
	if result.Added != 3 {
		t.Fatalf("Added: got %d, want 3", result.Added)
	}
	if result.Updated != 0 {
		t.Fatalf("Updated: got %d, want 0", result.Updated)
	}
	if result.Err != nil {
		t.Fatalf("Result.Err: %v", result.Err)
	}

	articles, err := d.ListArticles(feed.ID, 10)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("articles: got %d, want 3", len(articles))
	}
	// Articles are ordered by published_at DESC, so [0] is the newest.
	if articles[0].Title != "First post" {
		t.Fatalf("articles[0].Title: got %q, want %q", articles[0].Title, "First post")
	}
	if articles[0].URL != "http://example.org/2026/04/13/first" {
		t.Fatalf("articles[0].URL: got %q", articles[0].URL)
	}
}
```

**Step 3.3: Run test — expect failure**

```bash
go test ./internal/feed/... -run TestFetchOne_AtomHappyPath -v
```

Expected: compile error — `undefined: New`.

**Step 3.4: Implement the fetcher**

Create `rdr/internal/feed/fetcher.go`:

```go
package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/iRootPro/rdr/internal/db"
)

const userAgent = "rdr/0.1 (+https://github.com/iRootPro/rdr)"

type FetchResult struct {
	Feed    db.Feed
	Added   int
	Updated int
	Err     error
}

type Fetcher struct {
	db     *db.DB
	client *http.Client
	parser *gofeed.Parser
}

func New(d *db.DB) *Fetcher {
	return &Fetcher{
		db:     d,
		client: &http.Client{Timeout: 15 * time.Second},
		parser: gofeed.NewParser(),
	}
}

func (f *Fetcher) FetchOne(ctx context.Context, feed db.Feed) (FetchResult, error) {
	body, err := f.get(ctx, feed.URL)
	if err != nil {
		return FetchResult{}, err
	}
	defer body.Close()

	parsed, err := f.parser.Parse(body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("parse feed: %w", err)
	}

	result := FetchResult{Feed: feed}
	for _, item := range parsed.Items {
		article := mapItem(feed.ID, item)
		inserted, err := f.db.UpsertArticle(article)
		if err != nil {
			return FetchResult{}, fmt.Errorf("upsert: %w", err)
		}
		if inserted {
			result.Added++
		} else {
			result.Updated++
		}
	}
	return result, nil
}

func (f *Fetcher) get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func mapItem(feedID int64, item *gofeed.Item) db.Article {
	a := db.Article{
		FeedID:      feedID,
		Title:       item.Title,
		URL:         item.Link,
		Description: item.Description,
		Content:     item.Content,
	}
	if a.Content == "" {
		a.Content = item.Description
	}
	if a.Title == "" {
		a.Title = "(без заголовка)"
	}
	if item.PublishedParsed != nil {
		a.PublishedAt = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		a.PublishedAt = *item.UpdatedParsed
	} else {
		a.PublishedAt = time.Now().UTC()
	}
	return a
}
```

**Step 3.5: Run test — expect pass**

```bash
go test ./internal/feed/... -v
```

Expected: `TestFetchOne_AtomHappyPath` PASS.

**Step 3.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/feed/fetcher.go \
    rdr/internal/feed/fetcher_test.go \
    rdr/internal/feed/testdata/atom_feed.xml
git commit -m "feat(rdr): add feed.Fetcher with happy-path FetchOne"
```

---

## Task 4: FetchOne edge cases (TDD)

**Files:**
- Modify: `internal/feed/fetcher_test.go` (append tests)
- Create: `internal/feed/testdata/notitle_feed.xml`

**Step 4.1: Create the no-title fixture**

Create `rdr/internal/feed/testdata/notitle_feed.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>NoTitle Feed</title>
  <link href="http://example.org/"/>
  <updated>2026-04-13T12:00:00Z</updated>
  <id>urn:uuid:notitle</id>
  <entry>
    <link href="http://example.org/notitle/1"/>
    <id>urn:uuid:notitle-1</id>
    <updated>2026-04-13T11:00:00Z</updated>
    <published>2026-04-13T11:00:00Z</published>
    <summary>An entry without a title element</summary>
  </entry>
</feed>
```

**Step 4.2: Write the failing tests**

Append to `rdr/internal/feed/fetcher_test.go`:

```go
func TestFetchOne_IsIdempotent(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()
	feed, _ := d.UpsertFeed("Example", srv.URL)

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	result, err := f.FetchOne(context.Background(), feed)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if result.Added != 0 {
		t.Fatalf("Added on rerun: got %d, want 0", result.Added)
	}
	if result.Updated != 3 {
		t.Fatalf("Updated on rerun: got %d, want 3", result.Updated)
	}
}

func TestFetchOne_MalformedXMLReturnsError(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<not xml at all"))
	}))
	defer srv.Close()
	feed, _ := d.UpsertFeed("Bad", srv.URL)

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestFetchOne_HTTP500ReturnsError(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	feed, _ := d.UpsertFeed("Boom", srv.URL)

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err == nil {
		t.Fatal("expected http error, got nil")
	}
}

func TestFetchOne_EmptyTitleUsesFallback(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "notitle_feed.xml")
	defer srv.Close()
	feed, _ := d.UpsertFeed("NoTitle", srv.URL)

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("FetchOne: %v", err)
	}
	articles, _ := d.ListArticles(feed.ID, 10)
	if len(articles) != 1 {
		t.Fatalf("articles: got %d, want 1", len(articles))
	}
	if articles[0].Title != "(без заголовка)" {
		t.Fatalf("title fallback: got %q", articles[0].Title)
	}
}
```

**Step 4.3: Run tests — expect pass**

The implementation from Task 3 already covers all four cases (idempotency
via UpsertArticle's pre-check, parse error from gofeed, HTTP status check
in `get`, title fallback in `mapItem`). Run:

```bash
go test ./internal/feed/... -v
```

Expected: all 5 tests PASS. If anything fails, the implementation in
Task 3 was wrong — go fix `fetcher.go`, do **not** weaken the test.

**Step 4.4: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/feed/fetcher_test.go \
    rdr/internal/feed/testdata/notitle_feed.xml
git commit -m "test(rdr): cover FetchOne idempotency and error paths"
```

---

## Task 5: `FetchAll` with errgroup + per-feed error isolation (TDD)

**Files:**
- Modify: `internal/feed/fetcher.go` (add `FetchAll`)
- Modify: `internal/feed/fetcher_test.go` (append tests)

**Step 5.1: Write the failing test**

Append to `rdr/internal/feed/fetcher_test.go`:

```go
func TestFetchAll_ContinuesAfterPerFeedError(t *testing.T) {
	d := openTestDB(t)
	good := serveFixture(t, "atom_feed.xml")
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<broken"))
	}))
	defer bad.Close()

	goodFeed, _ := d.UpsertFeed("Good", good.URL)
	badFeed, _ := d.UpsertFeed("Bad", bad.URL)

	f := New(d)
	results, err := f.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results: got %d, want 2", len(results))
	}

	byID := map[int64]FetchResult{
		results[0].Feed.ID: results[0],
		results[1].Feed.ID: results[1],
	}
	if g := byID[goodFeed.ID]; g.Err != nil || g.Added != 3 {
		t.Fatalf("good feed: Err=%v Added=%d", g.Err, g.Added)
	}
	if b := byID[badFeed.ID]; b.Err == nil {
		t.Fatalf("bad feed: expected error, got Added=%d", b.Added)
	}

	// The good feed's articles must be in the database despite the bad feed failing.
	articles, _ := d.ListArticles(goodFeed.ID, 10)
	if len(articles) != 3 {
		t.Fatalf("good feed articles: got %d, want 3", len(articles))
	}
}
```

**Step 5.2: Run test — expect failure**

```bash
go test ./internal/feed/... -run TestFetchAll -v
```

Expected: compile error — `f.FetchAll undefined`.

**Step 5.3: Implement `FetchAll`**

Append to `rdr/internal/feed/fetcher.go`:

```go
func (f *Fetcher) FetchAll(ctx context.Context) ([]FetchResult, error) {
	feeds, err := f.db.ListFeeds()
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
	results := make([]FetchResult, len(feeds))
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 8)

	for i, feed := range feeds {
		i, feed := i, feed
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()

			r, err := f.FetchOne(gctx, feed)
			if err != nil {
				results[i] = FetchResult{Feed: feed, Err: err}
				return nil
			}
			results[i] = r
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return results, err
	}
	return results, nil
}
```

You also need to add `golang.org/x/sync/errgroup` to the imports of
`fetcher.go`.

**Step 5.4: Run all feed tests — expect pass**

```bash
go test ./internal/feed/... -v
```

Expected: all 6 tests PASS.

**Step 5.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/feed/fetcher.go rdr/internal/feed/fetcher_test.go
git commit -m "feat(rdr): add Fetcher.FetchAll with errgroup and per-feed error isolation"
```

---

## Task 6: Trim read articles after fetch (TDD)

The design says: after a successful fetch, the fetcher must call
`db.TrimArticles(feedID, max_articles_per_feed)` so the read articles
don't grow forever. The cap lives in the `settings` table (key
`max_articles_per_feed`, default `50`).

**Files:**
- Modify: `internal/feed/fetcher.go` (call Trim from FetchOne)
- Modify: `internal/feed/fetcher_test.go` (append test)

**Step 6.1: Write the failing test**

Append to `rdr/internal/feed/fetcher_test.go`:

```go
func TestFetchOne_TrimsReadArticlesToSettingCap(t *testing.T) {
	d := openTestDB(t)
	if err := d.SetSetting("max_articles_per_feed", "2"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	// Use a fixture-served feed so FetchOne completes successfully.
	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()
	feed, _ := d.UpsertFeed("Example", srv.URL)

	// Pre-populate 3 read articles older than the fixture's entries.
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		a := db.Article{
			FeedID:      feed.ID,
			Title:       "old",
			URL:         "https://old.example/" + string(rune('a'+i)),
			PublishedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	all, _ := d.ListArticles(feed.ID, 100)
	for _, a := range all {
		if err := d.MarkRead(a.ID); err != nil {
			t.Fatalf("MarkRead: %v", err)
		}
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("FetchOne: %v", err)
	}

	// After fetch we have 3 brand-new unread articles + at most 2 of the
	// 3 pre-seeded read articles (TrimArticles deletes oldest read first).
	// Trim only removes READ rows, so unread articles always survive.
	all, _ = d.ListArticles(feed.ID, 100)
	var unread, read int
	for _, a := range all {
		if a.ReadAt == nil {
			unread++
		} else {
			read++
		}
	}
	if unread != 3 {
		t.Fatalf("unread: got %d, want 3", unread)
	}
	if read != 2 {
		t.Fatalf("read: got %d, want 2 (cap=2 enforced on read rows)", read)
	}
}
```

Note: the existing `TrimArticles` semantics from Step 1 only delete read
articles, so this test pins both behaviors at once: trim is called AND the
existing trim semantics are respected.

**Step 6.2: Run test — expect failure**

```bash
go test ./internal/feed/... -run TestFetchOne_TrimsReadArticles -v
```

Expected: FAIL — currently the fetcher doesn't call TrimArticles, so all
3 pre-seeded read articles survive (read=3, want 2).

**Step 6.3: Implement trim-after-upsert**

Edit `rdr/internal/feed/fetcher.go`. Add a small helper at the bottom of
the file:

```go
func (f *Fetcher) maxArticlesPerFeed() int {
	const fallback = 50
	v, err := f.db.GetSetting("max_articles_per_feed")
	if err != nil || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
```

Add `"strconv"` to the import block.

Then update `FetchOne` — at the very end, **after** the upsert loop and
**before** `return result, nil`, add:

```go
if err := f.db.TrimArticles(feed.ID, f.maxArticlesPerFeed()); err != nil {
	return FetchResult{}, fmt.Errorf("trim: %w", err)
}
```

**Step 6.4: Run all feed tests — expect pass**

```bash
go test ./internal/feed/... -v
```

Expected: all 7 tests PASS.

**Step 6.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/feed/fetcher.go rdr/internal/feed/fetcher_test.go
git commit -m "feat(rdr): trim read articles to per-feed cap after fetch"
```

---

## Task 7: `--fetch` flag in `main.go`

This is the temporary CLI dev knob — the real entry point will be the TUI
in Step 4. Until then, `--fetch` lets us hit real feeds and confirm
end-to-end behavior.

**Files:**
- Modify: `main.go`

**Step 7.1: Update `main.go`**

Replace `rdr/main.go` with:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

func main() {
	doFetch := flag.Bool("fetch", false, "fetch all feeds before printing")
	flag.Parse()

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

	if *doFetch {
		fetcher := feed.New(database)
		results, err := fetcher.FetchAll(context.Background())
		if err != nil {
			log.Fatalf("fetch: %v", err)
		}
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("  ! %s: %v\n", r.Feed.Name, r.Err)
				continue
			}
			fmt.Printf("  ✓ %s: added=%d updated=%d\n", r.Feed.Name, r.Added, r.Updated)
		}
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

**Step 7.2: Build**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go build ./...
```

Expected: exits 0.

**Step 7.3: Smoke test — no flag (regression)**

```bash
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected: same output as Step 2 — 3 feeds with `unread: 0`, no fetch
output.

**Step 7.4: Smoke test — with `--fetch` (real network)**

```bash
RDR_HOME=./dev go run . --fetch
```

Expected: real HTTP fetch against HN/Go/Lobsters. Each feed prints a `✓`
line with `added=N updated=0` (first run). Then the feed list shows
`unread: N` reflecting the just-fetched articles. If a feed errors, you'll
see a `!` line — that's fine as long as at least one succeeds.

**Step 7.5: Smoke test — `--fetch` again (idempotent)**

```bash
RDR_HOME=./dev go run . --fetch
```

Expected: each `✓` line now shows `added=0 updated=N` (or close to it —
real feeds may have new posts between runs).

**Step 7.6: Inspect the database (optional sanity check)**

```bash
sqlite3 dev/rdr.db "SELECT feeds.name, COUNT(articles.id) FROM feeds LEFT JOIN articles ON articles.feed_id = feeds.id GROUP BY feeds.id;"
```

Expected: each feed has > 0 articles.

**Step 7.7: Cleanup and commit**

```bash
rm -rf dev
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/main.go
git commit -m "feat(rdr): add --fetch flag wiring fetcher into main"
```

---

## Task 8: Final verification

**Step 8.1: Full test run**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go test ./... -v
```

Expected: every test PASS. Step 2 had 22 tests; Step 3 adds:
- 1 from Task 2 (`TestUpsertArticle_ReturnsInsertedFlag`)
- 7 from Tasks 3–6 (happy path, idempotent, malformed, HTTP500, no-title, FetchAll, Trim)

= 30 tests total.

**Step 8.2: Build**

```bash
go build ./...
```

Expected: exits 0.

**Step 8.3: One more end-to-end smoke test**

```bash
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run . --fetch
RDR_HOME=./dev go run . --fetch
rm -rf dev
```

Expected: first run shows `added>0`, second run shows `updated>0 added~0`
(a couple of new posts on real feeds is normal). No panics, no errors
besides per-feed network hiccups.

---

## What's next

Step 3 leaves us with a working ingestion pipeline: feeds in YAML get
upserted, articles are pulled from the network in parallel, and old read
articles are trimmed. Step 4 starts the TUI: bubbletea Model, Tokyo Night
styles, FeedList + ArticleList split-pane. The `--fetch` flag dies in
Step 4 — fetching becomes an `Init()` command and a keystroke (`r`/`R`).
