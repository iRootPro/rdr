package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Article struct {
	ID          int64
	FeedID      int64
	FeedName    string // transient: populated only by cross-feed loaders
	Title       string
	URL         string
	Description string
	Content     string
	PublishedAt time.Time
	ReadAt      *time.Time
	StarredAt    *time.Time
	BookmarkedAt *time.Time
	CachedAt     *time.Time
	CachedBody  string
	CreatedAt   time.Time
}

// UpsertArticle inserts or updates an article and returns whether the row
// was newly inserted. The pre-check is racy under concurrent writers to
// the same (feed_id, url): the upsert itself is atomic, so rows stay
// consistent, but the inserted flag may be off — fetcher loops are
// per-feed serial so this is dormant in practice.
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

	// last_fetched_at is bumped on both INSERT and UPDATE so TrimArticles
	// can distinguish "still in this RSS response" from "rolled off the
	// feed and safe to prune". We bind a Go time.Time instead of using
	// SQLite's CURRENT_TIMESTAMP: the go-sqlite3 driver formats time
	// values with subseconds + timezone, while CURRENT_TIMESTAMP returns
	// seconds-only strings. Mixing the two breaks lexicographic
	// comparison (space < period) and causes TrimArticles to wrongly
	// prune freshly-upserted rows.
	fetchedAt := time.Now().UTC()
	_, err = d.sql.Exec(`
		INSERT INTO articles
			(feed_id, title, url, description, content, published_at, last_fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(feed_id, url) DO UPDATE SET
			title           = excluded.title,
			description     = excluded.description,
			content         = excluded.content,
			published_at    = excluded.published_at,
			last_fetched_at = excluded.last_fetched_at
	`, a.FeedID, a.Title, a.URL, a.Description, a.Content, a.PublishedAt, fetchedAt)
	if err != nil {
		return false, fmt.Errorf("upsert article: %w", err)
	}
	return inserted, nil
}

// ArticleFilter narrows a feed's article list. Kept as a DB-level enum so
// callers don't pass opaque booleans or magic strings.
type ArticleFilter int

const (
	FilterAll ArticleFilter = iota
	FilterUnread
	FilterStarred
)

func (d *DB) ListArticles(feedID int64, limit int) ([]Article, error) {
	return d.ListArticlesFiltered(feedID, FilterAll, limit)
}

// ListArticlesFiltered returns articles for a feed, narrowed by filter.
func (d *DB) ListArticlesFiltered(feedID int64, filter ArticleFilter, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 50
	}
	where := "WHERE feed_id = ?"
	args := []any{feedID}
	switch filter {
	case FilterUnread:
		where += " AND read_at IS NULL"
	case FilterStarred:
		where += " AND starred_at IS NOT NULL"
	}
	args = append(args, limit)

	rows, err := d.sql.Query(`
		SELECT id, feed_id, title, url, description, content,
		       published_at, read_at, starred_at, bookmarked_at, cached_at, cached_body, created_at
		FROM articles
		`+where+`
		ORDER BY published_at DESC, id DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var (
			a            Article
			desc, cont   sql.NullString
			readAt       sql.NullTime
			starredAt    sql.NullTime
			bookmarkedAt sql.NullTime
			cachedAt     sql.NullTime
			cachedBody   sql.NullString
		)
		if err := rows.Scan(
			&a.ID, &a.FeedID, &a.Title, &a.URL, &desc, &cont,
			&a.PublishedAt, &readAt, &starredAt, &bookmarkedAt, &cachedAt, &cachedBody, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		a.Description = desc.String
		a.Content = cont.String
		if readAt.Valid {
			t := readAt.Time
			a.ReadAt = &t
		}
		if starredAt.Valid {
			t := starredAt.Time
			a.StarredAt = &t
		}
		if bookmarkedAt.Valid {
			t := bookmarkedAt.Time
			a.BookmarkedAt = &t
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

// ListAllArticles returns articles across all feeds (no feed_id filter),
// joined with the feed name. Used by smart folders — callers then apply
// in-memory query filtering on the result set.
func (d *DB) ListAllArticles(limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 2000
	}
	rows, err := d.sql.Query(`
		SELECT a.id, a.feed_id, f.name, a.title, a.url, a.description, a.content,
		       a.published_at, a.read_at, a.starred_at, a.bookmarked_at, a.cached_at, a.cached_body, a.created_at
		FROM articles a
		JOIN feeds f ON f.id = a.feed_id
		ORDER BY a.published_at DESC, a.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var (
			a            Article
			desc, cont   sql.NullString
			readAt       sql.NullTime
			starredAt    sql.NullTime
			bookmarkedAt sql.NullTime
			cachedAt     sql.NullTime
			cachedBody   sql.NullString
		)
		if err := rows.Scan(
			&a.ID, &a.FeedID, &a.FeedName, &a.Title, &a.URL, &desc, &cont,
			&a.PublishedAt, &readAt, &starredAt, &bookmarkedAt, &cachedAt, &cachedBody, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		a.Description = desc.String
		a.Content = cont.String
		if readAt.Valid {
			t := readAt.Time
			a.ReadAt = &t
		}
		if starredAt.Valid {
			t := starredAt.Time
			a.StarredAt = &t
		}
		if bookmarkedAt.Valid {
			t := bookmarkedAt.Time
			a.BookmarkedAt = &t
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

// MarkFeedRead marks every unread article in a feed as read in a single
// UPDATE and returns the affected row count.
func (d *DB) MarkFeedRead(feedID int64) (int, error) {
	res, err := d.sql.Exec(
		`UPDATE articles SET read_at = ? WHERE feed_id = ? AND read_at IS NULL`,
		time.Now().UTC(), feedID,
	)
	if err != nil {
		return 0, fmt.Errorf("mark feed read: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// MarkUnread clears the read state of a single article. Idempotent.
func (d *DB) MarkUnread(articleID int64) error {
	_, err := d.sql.Exec(
		`UPDATE articles SET read_at = NULL WHERE id = ?`,
		articleID,
	)
	return err
}

// BulkMarkRead marks many articles read in one UPDATE. No-op on empty
// input. Idempotent for already-read rows.
func (d *DB) BulkMarkRead(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query, args := bulkIDQuery(
		`UPDATE articles SET read_at = ? WHERE read_at IS NULL AND id IN`,
		ids, time.Now().UTC(),
	)
	_, err := d.sql.Exec(query, args...)
	return err
}

// BulkMarkUnread clears read_at on many articles in one UPDATE.
func (d *DB) BulkMarkUnread(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query, args := bulkIDQuery(
		`UPDATE articles SET read_at = NULL WHERE id IN`,
		ids,
	)
	_, err := d.sql.Exec(query, args...)
	return err
}

// BulkSetStarred sets starred_at to now() or NULL on many articles.
func (d *DB) BulkSetStarred(ids []int64, starred bool) error {
	if len(ids) == 0 {
		return nil
	}
	if starred {
		query, args := bulkIDQuery(
			`UPDATE articles SET starred_at = ? WHERE starred_at IS NULL AND id IN`,
			ids, time.Now().UTC(),
		)
		_, err := d.sql.Exec(query, args...)
		return err
	}
	query, args := bulkIDQuery(
		`UPDATE articles SET starred_at = NULL WHERE id IN`,
		ids,
	)
	_, err := d.sql.Exec(query, args...)
	return err
}

// bulkIDQuery appends a parenthesized placeholder list "(?, ?, ...)" to
// the given prefix and returns the full query plus the combined argument
// slice. leadingArgs come first (e.g. the timestamp for UPDATE ... SET x=?).
func bulkIDQuery(prefix string, ids []int64, leadingArgs ...any) (string, []any) {
	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}
	query := prefix + " (" + strings.Join(placeholders, ",") + ")"
	args := make([]any, 0, len(leadingArgs)+len(ids))
	args = append(args, leadingArgs...)
	for _, id := range ids {
		args = append(args, id)
	}
	return query, args
}

// ToggleStar flips the starred state of an article: nil ↔ now(). Returns
// whether the article is now starred.
func (d *DB) ToggleStar(articleID int64) (bool, error) {
	var starredAt sql.NullTime
	err := d.sql.QueryRow(`SELECT starred_at FROM articles WHERE id = ?`, articleID).Scan(&starredAt)
	if err != nil {
		return false, fmt.Errorf("read starred_at: %w", err)
	}
	if starredAt.Valid {
		_, err := d.sql.Exec(`UPDATE articles SET starred_at = NULL WHERE id = ?`, articleID)
		return false, err
	}
	_, err = d.sql.Exec(`UPDATE articles SET starred_at = ? WHERE id = ?`, time.Now().UTC(), articleID)
	return true, err
}

// ToggleBookmark flips the bookmarked (read-later) state of an article.
func (d *DB) ToggleBookmark(articleID int64) (bool, error) {
	var bookmarkedAt sql.NullTime
	err := d.sql.QueryRow(`SELECT bookmarked_at FROM articles WHERE id = ?`, articleID).Scan(&bookmarkedAt)
	if err != nil {
		return false, fmt.Errorf("read bookmarked_at: %w", err)
	}
	if bookmarkedAt.Valid {
		_, err := d.sql.Exec(`UPDATE articles SET bookmarked_at = NULL WHERE id = ?`, articleID)
		return false, err
	}
	_, err = d.sql.Exec(`UPDATE articles SET bookmarked_at = ? WHERE id = ?`, time.Now().UTC(), articleID)
	return true, err
}

// BulkSetBookmarked sets bookmarked_at to now() or NULL on many articles.
func (d *DB) BulkSetBookmarked(ids []int64, bookmark bool) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var query string
	var args []any
	if bookmark {
		query, args = bulkIDQuery(
			`UPDATE articles SET bookmarked_at = ? WHERE bookmarked_at IS NULL AND id IN`,
			ids, time.Now().UTC())
	} else {
		query, args = bulkIDQuery(
			`UPDATE articles SET bookmarked_at = NULL WHERE id IN`,
			ids)
	}
	res, err := d.sql.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (d *DB) MarkRead(articleID int64) error {
	_, err := d.sql.Exec(
		`UPDATE articles SET read_at = ? WHERE id = ? AND read_at IS NULL`,
		time.Now().UTC(), articleID,
	)
	return err
}

// SaveLibraryURL inserts a saved URL into the Library feed (or updates
// the placeholder title on duplicate, preserving read/star/bookmark
// state). Returns the article ID and whether it was newly inserted.
// Used by the Add-URL modal: caller passes a fast placeholder title
// (e.g. host) and updates it later via UpdateLibraryFetched once the
// background readability fetch returns the real <title>.
func (d *DB) SaveLibraryURL(libraryFeedID int64, url, placeholderTitle string) (int64, bool, error) {
	now := time.Now().UTC()
	tx, err := d.sql.Begin()
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	err = tx.QueryRow(
		`SELECT id FROM articles WHERE feed_id = ? AND url = ?`,
		libraryFeedID, url,
	).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, false, fmt.Errorf("check library article: %w", err)
	}

	if existingID != 0 {
		// Bump last_fetched_at so Trim safety logic stays consistent
		// even though Library is never trimmed today.
		if _, err := tx.Exec(
			`UPDATE articles SET last_fetched_at = ? WHERE id = ?`,
			now, existingID,
		); err != nil {
			return 0, false, fmt.Errorf("touch library article: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return 0, false, err
		}
		return existingID, false, nil
	}

	res, err := tx.Exec(`
		INSERT INTO articles
			(feed_id, title, url, description, content, published_at, last_fetched_at)
		VALUES (?, ?, ?, '', '', ?, ?)
	`, libraryFeedID, placeholderTitle, url, now, now)
	if err != nil {
		return 0, false, fmt.Errorf("insert library article: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// UpdateLibraryFetched stores the result of an async readability fetch:
// the extracted page title (if non-empty) replaces the placeholder, and
// the Markdown body is cached so the reader can show it immediately on
// open. Title is only overwritten when extraction succeeded — empty
// titles leave the placeholder intact.
func (d *DB) UpdateLibraryFetched(id int64, title, body string) error {
	now := time.Now().UTC()
	if title != "" {
		_, err := d.sql.Exec(
			`UPDATE articles SET title = ?, cached_body = ?, cached_at = ? WHERE id = ?`,
			title, body, now, id,
		)
		if err != nil {
			return fmt.Errorf("update library article: %w", err)
		}
		return nil
	}
	_, err := d.sql.Exec(
		`UPDATE articles SET cached_body = ?, cached_at = ? WHERE id = ?`,
		body, now, id,
	)
	if err != nil {
		return fmt.Errorf("update library article body: %w", err)
	}
	return nil
}

// DeleteArticle removes a single article by id. Used by the Library
// pane to discard saved URLs the user no longer wants — TrimArticles
// skips the system feed, so without this the table grows forever.
func (d *DB) DeleteArticle(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete article: %w", err)
	}
	return nil
}

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

type SearchItem struct {
	ArticleID    int64
	FeedID       int64
	FeedName     string
	Title        string
	URL          string
	Description  string
	CachedBody   string
	PublishedAt  time.Time
	ReadAt       *time.Time
	StarredAt    *time.Time
	BookmarkedAt *time.Time
}

func (d *DB) SearchArticles(limit int) ([]SearchItem, error) {
	if limit <= 0 {
		limit = 2000
	}
	rows, err := d.sql.Query(`
		SELECT a.id, a.feed_id, f.name, a.title, a.url,
		       a.description, a.cached_body, a.published_at, a.read_at, a.starred_at, a.bookmarked_at
		FROM articles a
		JOIN feeds f ON f.id = a.feed_id
		ORDER BY a.published_at DESC, a.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SearchItem
	for rows.Next() {
		var (
			item         SearchItem
			desc         sql.NullString
			cachedBody   sql.NullString
			readAt       sql.NullTime
			starredAt    sql.NullTime
			bookmarkedAt sql.NullTime
		)
		if err := rows.Scan(
			&item.ArticleID, &item.FeedID, &item.FeedName, &item.Title, &item.URL,
			&desc, &cachedBody, &item.PublishedAt, &readAt, &starredAt, &bookmarkedAt,
		); err != nil {
			return nil, err
		}
		item.Description = desc.String
		item.CachedBody = cachedBody.String
		if readAt.Valid {
			t := readAt.Time
			item.ReadAt = &t
		}
		if starredAt.Valid {
			t := starredAt.Time
			item.StarredAt = &t
		}
		if bookmarkedAt.Valid {
			t := bookmarkedAt.Time
			item.BookmarkedAt = &t
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// TrimArticles prunes old read articles in a feed once its total count
// exceeds max. An article is eligible for deletion only when ALL of:
//
//   - read_at IS NOT NULL (unread is always kept)
//   - starred_at IS NULL  (star is an explicit user "save")
//   - bookmarked_at IS NULL (read-later is an explicit user "save")
//   - read_at < readCutoff (recently-read articles get a grace window so
//     a stray `x` keystroke doesn't immediately destroy the row)
//   - last_fetched_at IS NULL OR last_fetched_at < fetchCutoff (anything
//     still in the current RSS response is protected — otherwise marking
//     read right before a refresh would delete it and the next fetch
//     would re-insert it as unread)
func (d *DB) TrimArticles(feedID int64, max int, fetchCutoff, readCutoff time.Time) error {
	if max <= 0 {
		return nil
	}
	_, err := d.sql.Exec(`
		DELETE FROM articles
		WHERE id IN (
			SELECT id FROM articles
			WHERE feed_id = ?
			  AND read_at IS NOT NULL
			  AND read_at < ?
			  AND starred_at IS NULL
			  AND bookmarked_at IS NULL
			  AND (last_fetched_at IS NULL OR last_fetched_at < ?)
			ORDER BY published_at ASC, id ASC
			LIMIT MAX(0, (
				SELECT COUNT(*) FROM articles WHERE feed_id = ?
			) - ?)
		)
	`, feedID, readCutoff, fetchCutoff, feedID, max)
	return err
}
