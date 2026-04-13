package db

import (
	"database/sql"
	"errors"
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
