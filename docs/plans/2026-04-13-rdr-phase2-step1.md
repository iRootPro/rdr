# rdr Phase 2 — Step 1: Full Article Reader

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Press `f` in the Reader to fetch the full article via
`go-shiori/go-readability`, convert HTML → Markdown, render with
`charmbracelet/glamour`, and cache the result in `articles.cached_body`
so repeat opens are instant. Step 2 adds Kitty Graphics for inline
images. This step delivers clean readable full articles with dark
Tokyo-Night-compatible styling.

**Architecture:**
- `internal/feed/reader.go` — new file: `FetchFull(ctx, url) (string, error)`
  does HTTP GET → go-readability → html-to-markdown. Returns markdown.
- `internal/db/articles.go` — new `CacheArticle(id int64, body string) error`
  sets `cached_body` + `cached_at` for a row.
- `internal/ui/` — `f` key in reader focus triggers `fetchFullCmd`.
  Spinner indicates loading. On success, the reader content is replaced
  with glamour-rendered markdown, and the body is cached via
  `CacheArticle`. If the article already has `cached_body`, the reader
  uses that instead of the RSS content (no network call).

**Tech Stack:** `github.com/go-shiori/go-readability`,
`github.com/JohannesKaufmann/html-to-markdown/v2`,
`github.com/charmbracelet/glamour`.

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

---

## Task 1: `db.CacheArticle` (TDD)

**Files:**
- Modify: `internal/db/articles.go` (new method)
- Modify: `internal/db/articles_test.go` (new test)

**Step 1.1: Write the failing test**

Append to `articles_test.go`:

```go
func TestCacheArticle_SetsCachedBodyAndCachedAt(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	a := newArticle(f.ID, "https://a.example/1", "v1", time.Now().UTC())
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, _ := d.ListArticles(f.ID, 10)
	if len(list) != 1 {
		t.Fatalf("want 1 article, got %d", len(list))
	}
	id := list[0].ID
	if list[0].CachedBody != "" || list[0].CachedAt != nil {
		t.Fatalf("pre-cache state wrong: %+v", list[0])
	}

	if err := d.CacheArticle(id, "# Hello\n\nBody"); err != nil {
		t.Fatalf("CacheArticle: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].CachedBody != "# Hello\n\nBody" {
		t.Fatalf("cached_body: %q", list[0].CachedBody)
	}
	if list[0].CachedAt == nil {
		t.Fatalf("cached_at is nil after CacheArticle")
	}
}
```

**Step 1.2: Run — expect failure**

```bash
go test ./internal/db/... -run TestCacheArticle -v
```

Expected: `d.CacheArticle undefined`.

**Step 1.3: Implement**

Add to `articles.go`:

```go
func (d *DB) CacheArticle(id int64, body string) error {
	_, err := d.sql.Exec(
		`UPDATE articles SET cached_body = ?, cached_at = ? WHERE id = ?`,
		body, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("cache article: %w", err)
	}
	return nil
}
```

**Step 1.4: Run — expect pass**

```bash
go test ./internal/db/... -v
```

Expected: all db tests pass (13 existing + 1 new = 14).

**Step 1.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/articles.go rdr/internal/db/articles_test.go
git commit -m "feat(rdr): add db.CacheArticle for full-article caching"
```

---

## Task 2: `feed.FetchFull` (TDD)

**Files:**
- Modify: `go.mod` / `go.sum` (via `go get`)
- Modify: `internal/feed/fetcher.go` (new `FetchFull` method)
- Modify: `internal/feed/fetcher_test.go` (new test)
- Create: `internal/feed/testdata/article.html`

**Step 2.1: Install deps**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go get github.com/go-shiori/go-readability
go get github.com/JohannesKaufmann/html-to-markdown/v2
```

**Step 2.2: Create HTML fixture**

Create `rdr/internal/feed/testdata/article.html`:

```html
<!DOCTYPE html>
<html>
<head><title>Example Article</title></head>
<body>
<header>Site header noise</header>
<nav>Menu</nav>
<article>
  <h1>The Real Title</h1>
  <p>This is the first paragraph. It has <strong>bold</strong> text and <em>italics</em>.</p>
  <p>This is the second paragraph with a <a href="https://example.com">link</a>.</p>
  <ul>
    <li>List item one</li>
    <li>List item two</li>
  </ul>
</article>
<footer>Footer noise</footer>
</body>
</html>
```

**Step 2.3: Write the failing test**

Append to `fetcher_test.go`:

```go
func TestFetchFull_ExtractsAndConvertsToMarkdown(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		body, _ := os.ReadFile(filepath.Join("testdata", "article.html"))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f := New(d)
	md, err := f.FetchFull(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFull: %v", err)
	}
	if md == "" {
		t.Fatal("empty markdown")
	}
	// Content checks — readability should strip header/nav/footer and keep the article body.
	for _, want := range []string{"first paragraph", "second paragraph", "List item one"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "Site header noise") || strings.Contains(md, "Footer noise") {
		t.Fatalf("readability did not strip chrome:\n%s", md)
	}
}
```

Add `"strings"` to imports if missing.

**Step 2.4: Run — expect failure**

```bash
go test ./internal/feed/... -run TestFetchFull -v
```

Expected: `f.FetchFull undefined`.

**Step 2.5: Implement `FetchFull`**

Append to `fetcher.go`:

```go
func (f *Fetcher) FetchFull(ctx context.Context, articleURL string) (string, error) {
	body, err := f.get(ctx, articleURL)
	if err != nil {
		return "", err
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	parsed, err := url.Parse(articleURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	article, err := readability.FromReader(bytes.NewReader(raw), parsed)
	if err != nil {
		return "", fmt.Errorf("readability: %w", err)
	}

	md, err := htmltomarkdown.ConvertString(article.Content)
	if err != nil {
		return "", fmt.Errorf("html to markdown: %w", err)
	}
	return md, nil
}
```

Add imports:
```go
"bytes"
"io"
"net/url"

readability "github.com/go-shiori/go-readability"
htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
```

**Step 2.6: Run — expect pass**

```bash
go test ./internal/feed/... -v
```

Expected: 9 feed tests pass (8 existing + 1 new).

**Step 2.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum \
    rdr/internal/feed/fetcher.go \
    rdr/internal/feed/fetcher_test.go \
    rdr/internal/feed/testdata/article.html
git commit -m "feat(rdr): add Fetcher.FetchFull via go-readability + html-to-markdown"
```

---

## Task 3: UI wiring — `f` key, glamour render, cache

**Files:**
- Modify: `go.mod` (glamour)
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/messages.go`
- Modify: `internal/ui/keys.go`
- Modify: `internal/ui/reader.go` (glamour render path)

**Step 3.1: Install glamour**

```bash
go get github.com/charmbracelet/glamour
```

**Step 3.2: Add FullArticle key**

`keys.go`:

```go
type keyMap struct {
	// ... existing ...
	FullArticle key.Binding
}

// in defaultKeys():
FullArticle: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "full article")),
```

Add to `FullHelp`:
```go
{k.RefreshOne, k.RefreshAll, k.OpenURL, k.FullArticle},
```

**Step 3.3: Add message types**

`messages.go`:

```go
type fullArticleLoadedMsg struct {
	articleID int64
	markdown  string
}
```

**Step 3.4: Add `fetchFullCmd`**

`model.go`:

```go
func fetchFullCmd(f *feed.Fetcher, d *db.DB, articleID int64, url string) tea.Cmd {
	return func() tea.Msg {
		md, err := f.FetchFull(context.Background(), url)
		if err != nil {
			return errMsg{err}
		}
		if err := d.CacheArticle(articleID, md); err != nil {
			return errMsg{err}
		}
		return fullArticleLoadedMsg{articleID: articleID, markdown: md}
	}
}
```

**Step 3.5: Handle `f` key in reader focus**

In `Update`:

```go
case key.Matches(msg, m.keys.FullArticle):
	if m.focus == focusReader && m.readerArt != nil && m.readerArt.URL != "" {
		m.fetching = true
		m.status = "loading full…"
		return m, tea.Batch(
			fetchFullCmd(m.fetcher, m.db, m.readerArt.ID, m.readerArt.URL),
			m.spin.Tick,
		)
	}
	return m, nil
```

Handle `fullArticleLoadedMsg`:

```go
case fullArticleLoadedMsg:
	m.fetching = false
	m.status = "full article"
	if m.readerArt != nil && m.readerArt.ID == msg.articleID {
		m.readerArt.CachedBody = msg.markdown
		now := time.Now().UTC()
		m.readerArt.CachedAt = &now
		feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
		m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4))
		m.reader.GotoTop()
	}
	return m, nil
```

Add `"time"` import to `model.go`.

**Step 3.6: Render cached body via glamour**

Update `buildReaderContent` in `reader.go`. When `a.CachedBody != ""`,
render that with glamour instead of falling back to stripHTML on
`a.Content`/`a.Description`:

```go
body := ""
if a.CachedBody != "" {
	rendered, err := glamour.Render(a.CachedBody, "dark")
	if err == nil {
		body = strings.TrimRight(rendered, "\n")
	}
}
if body == "" {
	body = stripHTML(a.Content)
	if body == "" {
		body = stripHTML(a.Description)
	}
	if body == "" {
		body = "(no content)"
	}
	body = readerBody.Render(wrap(body, width))
}
b.WriteString(body)
```

Note: glamour output is already styled with ANSI, so don't wrap it in
`readerBody.Render`. The stripHTML fallback still goes through the
wrap+style path.

Also update the `[f] load full article` hint to only show when no cached
body exists:

```go
if a.CachedBody == "" {
	b.WriteString("\n\n")
	b.WriteString(readerHint.Render("[f] load full article"))
}
```

Add import:
```go
"github.com/charmbracelet/glamour"
```

Glamour needs a width. Pass `width` through to the render call:

```go
renderer, err := glamour.NewTermRenderer(
	glamour.WithStandardStyle("dark"),
	glamour.WithWordWrap(width),
)
if err == nil {
	if rendered, err := renderer.Render(a.CachedBody); err == nil {
		body = strings.TrimRight(rendered, "\n")
	}
}
```

**Step 3.7: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Expected end-to-end:
1. Open an article in the reader — see the RSS content (stripHTML path)
2. Press `f` — status bar shows spinner + `loading full…`
3. A few seconds later — reader re-renders with glamour-styled Markdown
   (headings in color, links underlined, code blocks highlighted)
4. Close with `esc`, reopen the same article — the full body renders
   immediately from cache, no network call
5. The `[f]` hint is gone for articles that already have cached content

**Step 3.8: Final test run**

```bash
go test ./... -race
go build ./...
```

Expected: all tests still pass (32 total — 31 from Phase 1 + 1 new db
test + 1 new feed test = 33 actually).

**Step 3.9: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum \
    rdr/internal/ui/model.go \
    rdr/internal/ui/messages.go \
    rdr/internal/ui/keys.go \
    rdr/internal/ui/reader.go
git commit -m "feat(rdr): add f key to load full article with glamour render"
```

---

## What's next

Phase 2 Step 1 delivers real article rendering with caching. Step 2 adds
Kitty Graphics Protocol for inline images in the glamour-rendered body.
Step 3 adds a settings TUI for managing feeds (add/remove/rename) so
config.yaml becomes optional.
