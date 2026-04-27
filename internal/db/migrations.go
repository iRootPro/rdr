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
	// 004: smart folders
	`
	CREATE TABLE smart_folders (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL,
		query       TEXT NOT NULL,
		position    INTEGER NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX idx_smart_folders_position ON smart_folders(position, id);
	`,
	// 005: track last_fetched_at so Trim does not delete articles that
	// are still in the feed's current RSS response. Backfill existing
	// rows with NOW so the first fetch after upgrade doesn't nuke them.
	`
	ALTER TABLE articles ADD COLUMN last_fetched_at DATETIME;
	UPDATE articles SET last_fetched_at = CURRENT_TIMESTAMP;
	CREATE INDEX idx_articles_last_fetched_at ON articles(last_fetched_at);
	`,
	// 006: read-later bookmarks
	`
	ALTER TABLE articles ADD COLUMN bookmarked_at DATETIME;
	CREATE INDEX idx_articles_bookmarked_at ON articles(bookmarked_at) WHERE bookmarked_at IS NOT NULL;
	`,
	// 007: Library system feed for arbitrary saved URLs. Sentinel URL
	// 'internal://library' lets us tell user-subscribed feeds apart from
	// this one in ListFeeds / sync / OPML export. position=-1 keeps it
	// out of the user-position sequence so UpsertFeed's MAX(position)+1
	// stays consistent.
	`
	INSERT OR IGNORE INTO feeds (name, url, position, category)
	VALUES ('Library', 'internal://library', -1, '');
	`,

	// 008: HTTP Basic Auth credentials for private RSS feeds
	`
	ALTER TABLE feeds ADD COLUMN username TEXT;
	ALTER TABLE feeds ADD COLUMN password TEXT;
	`,
}
