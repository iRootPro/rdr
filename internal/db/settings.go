package db

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
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

const (
	settingKeyLanguage          = "language"
	settingKeyShowImages        = "show_images"
	settingKeySortField         = "sort_field"
	settingKeySortReverse       = "sort_reverse"
	settingKeyShowPreview       = "show_preview"
	settingKeyTheme             = "theme"
	settingKeyReadRetentionDays = "read_retention_days"
)

// defaultReadRetentionDays is the fallback value of read_retention_days
// for fresh installs (key unset in DB). 90 days gives back-search room
// for seasonal content without letting the DB grow unboundedly on
// high-volume feeds — max_articles_per_feed still caps per-feed totals.
const defaultReadRetentionDays = 90

func (d *DB) GetLanguage() (string, error) {
	return d.GetSetting(settingKeyLanguage)
}

func (d *DB) SetLanguage(lang string) error {
	return d.SetSetting(settingKeyLanguage, lang)
}

func (d *DB) GetShowImages() (bool, error) {
	v, err := d.GetSetting(settingKeyShowImages)
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

func (d *DB) SetShowImages(v bool) error {
	return d.SetSetting(settingKeyShowImages, boolToStr(v))
}

func (d *DB) GetSortField() (string, error) {
	return d.GetSetting(settingKeySortField)
}

func (d *DB) SetSortField(v string) error {
	return d.SetSetting(settingKeySortField, v)
}

func (d *DB) GetSortReverse() (bool, error) {
	v, err := d.GetSetting(settingKeySortReverse)
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

func (d *DB) SetSortReverse(v bool) error {
	return d.SetSetting(settingKeySortReverse, boolToStr(v))
}

// GetShowPreview returns whether the inline article preview popup is
// enabled. Defaults to true when the setting has never been written —
// the feature is on-by-default so first-time users see it without
// hunting through settings.
func (d *DB) GetShowPreview() (bool, error) {
	v, err := d.GetSetting(settingKeyShowPreview)
	if err != nil {
		return false, err
	}
	if v == "" {
		return true, nil
	}
	return v == "true", nil
}

func (d *DB) SetShowPreview(v bool) error {
	return d.SetSetting(settingKeyShowPreview, boolToStr(v))
}

func (d *DB) GetTheme() (string, error) {
	return d.GetSetting(settingKeyTheme)
}

func (d *DB) SetTheme(name string) error {
	return d.SetSetting(settingKeyTheme, name)
}

const (
	settingKeyRefreshInterval  = "refresh_interval_minutes"
	settingKeyAfterSyncCommands = "after_sync_commands"
)

func (d *DB) GetRefreshInterval() (int, error) {
	v, err := d.GetSetting(settingKeyRefreshInterval)
	if err != nil || v == "" {
		return 0, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func (d *DB) SetRefreshInterval(minutes int) error {
	return d.SetSetting(settingKeyRefreshInterval, strconv.Itoa(minutes))
}

func (d *DB) GetAfterSyncCommands() ([]string, error) {
	v, err := d.GetSetting(settingKeyAfterSyncCommands)
	if err != nil || v == "" {
		return nil, err
	}
	return strings.Split(v, "\n"), nil
}

func (d *DB) SetAfterSyncCommands(cmds []string) error {
	return d.SetSetting(settingKeyAfterSyncCommands, strings.Join(cmds, "\n"))
}

const (
	settingKeyAIProvider = "ai_provider"
	settingKeyAIEndpoint = "ai_endpoint"
	settingKeyAIKey      = "ai_api_key"
	settingKeyAIModel    = "ai_model"
)

func (d *DB) GetAIProvider() (string, error) { return d.GetSetting(settingKeyAIProvider) }
func (d *DB) SetAIProvider(v string) error   { return d.SetSetting(settingKeyAIProvider, v) }

func (d *DB) GetAIEndpoint() (string, error) { return d.GetSetting(settingKeyAIEndpoint) }
func (d *DB) SetAIEndpoint(v string) error   { return d.SetSetting(settingKeyAIEndpoint, v) }

func (d *DB) GetAIKey() (string, error) { return d.GetSetting(settingKeyAIKey) }
func (d *DB) SetAIKey(v string) error   { return d.SetSetting(settingKeyAIKey, v) }

func (d *DB) GetAIModel() (string, error) { return d.GetSetting(settingKeyAIModel) }
func (d *DB) SetAIModel(v string) error   { return d.SetSetting(settingKeyAIModel, v) }

// GetReadRetentionDays returns how many days read articles are kept by
// TrimArticles before age-based deletion. 0 means keep indefinitely (by
// age; max_articles_per_feed still caps the per-feed count). When the
// key is unset, returns defaultReadRetentionDays without writing it —
// we don't want a silent auto-write to obscure "never configured" from
// "configured to 90".
func (d *DB) GetReadRetentionDays() (int, error) {
	v, err := d.GetSetting(settingKeyReadRetentionDays)
	if err != nil {
		return defaultReadRetentionDays, err
	}
	if v == "" {
		return defaultReadRetentionDays, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return defaultReadRetentionDays, nil
	}
	return n, nil
}

// SetReadRetentionDays persists the retention window (days). Passing 0
// disables age-based deletion.
func (d *DB) SetReadRetentionDays(days int) error {
	if days < 0 {
		days = 0
	}
	return d.SetSetting(settingKeyReadRetentionDays, strconv.Itoa(days))
}

func boolToStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
