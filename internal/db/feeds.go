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
	Category    string
	Position    int
	CreatedAt   time.Time
	UnreadCount int
}

// UpsertFeed inserts or updates a feed by URL. Category is overwritten
// on conflict so yaml edits take effect on next sync.
func (d *DB) UpsertFeed(name, url, category string) (Feed, error) {
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
		INSERT INTO feeds (name, url, category, position) VALUES (?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			name = excluded.name,
			category = excluded.category
	`, name, url, category, nextPos)
	if err != nil {
		return Feed{}, fmt.Errorf("upsert: %w", err)
	}

	var f Feed
	row := tx.QueryRow(`
		SELECT id, name, url, category, position, created_at
		FROM feeds WHERE url = ?
	`, url)
	if err := row.Scan(&f.ID, &f.Name, &f.URL, &f.Category, &f.Position, &f.CreatedAt); err != nil {
		return Feed{}, fmt.Errorf("read back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Feed{}, err
	}
	return f, nil
}

func (d *DB) ListFeeds() ([]Feed, error) {
	rows, err := d.sql.Query(`
		SELECT f.id, f.name, f.url, f.category, f.position, f.created_at,
		       COUNT(CASE WHEN a.id IS NOT NULL AND a.read_at IS NULL THEN 1 END)
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
			&f.ID, &f.Name, &f.URL, &f.Category, &f.Position, &f.CreatedAt, &f.UnreadCount,
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
		SELECT id, name, url, category, position, created_at
		FROM feeds WHERE url = ?
	`, url).Scan(&f.ID, &f.Name, &f.URL, &f.Category, &f.Position, &f.CreatedAt)
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

func (d *DB) RenameFeed(id int64, name string) error {
	_, err := d.sql.Exec(`UPDATE feeds SET name = ? WHERE id = ?`, name, id)
	return err
}

// SetFeedCategory moves a feed into (or out of) a category. Empty
// string means "uncategorized". Used by :categorymove command.
func (d *DB) SetFeedCategory(id int64, category string) error {
	_, err := d.sql.Exec(`UPDATE feeds SET category = ? WHERE id = ?`, category, id)
	return err
}

// RenameCategory renames one category to another across all feeds. The
// new name may be empty, which moves matching feeds into the "Other"
// bucket. No-op when no feeds match.
func (d *DB) RenameCategory(oldName, newName string) error {
	_, err := d.sql.Exec(
		`UPDATE feeds SET category = ? WHERE category = ?`,
		newName, oldName,
	)
	return err
}

// DeleteCategory empties the category of every feed that currently has
// it, moving them into the "Other" bucket. Thin alias around
// RenameCategory(name, "") that makes the caller's intent explicit.
func (d *DB) DeleteCategory(name string) error {
	return d.RenameCategory(name, "")
}
