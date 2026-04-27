package config

import (
	"fmt"

	"github.com/iRootPro/rdr/internal/db"
)

// Sync upserts cfg.Feeds into the database. Feeds present in the DB but
// absent from cfg are intentionally NOT removed — manual deletion is the
// user's job until the Settings TUI ships.
func Sync(d *db.DB, cfg *Config) error {
	if cfg == nil {
		return nil
	}
	for _, e := range cfg.Feeds {
		if _, err := d.UpsertFeed(e.Name, e.URL, e.Category, e.Username, e.Password); err != nil {
			return fmt.Errorf("sync feed %q: %w", e.URL, err)
		}
	}
	return nil
}
