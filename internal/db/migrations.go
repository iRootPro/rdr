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
	// 002: article starring
	`
	ALTER TABLE articles ADD COLUMN starred_at DATETIME;
	CREATE INDEX idx_articles_starred_at ON articles(starred_at) WHERE starred_at IS NOT NULL;
	`,
	// 003: feed categories
	`
	ALTER TABLE feeds ADD COLUMN category TEXT NOT NULL DEFAULT '';
	CREATE INDEX idx_feeds_category ON feeds(category);
	`,
}
