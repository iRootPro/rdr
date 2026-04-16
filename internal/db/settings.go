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
	settingKeyLanguage    = "language"
	settingKeyShowImages  = "show_images"
	settingKeySortField   = "sort_field"
	settingKeySortReverse = "sort_reverse"
	settingKeyShowPreview = "show_preview"
	settingKeyTheme       = "theme"
)

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

func boolToStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
