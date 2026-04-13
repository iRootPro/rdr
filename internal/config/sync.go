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
