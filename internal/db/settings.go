package db

import (
	"database/sql"
	"errors"
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
