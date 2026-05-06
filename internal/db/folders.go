package db

import "fmt"

// ListFolders returns explicit regular folders plus any non-empty feed
// categories that may have been created by older versions or config sync.
func (d *DB) ListFolders() ([]string, error) {
	rows, err := d.sql.Query(`
		SELECT name FROM (
			SELECT name, MIN(position) AS position FROM (
				SELECT name, position FROM folders
				UNION ALL
				SELECT DISTINCT category AS name, 1000000 AS position
				FROM feeds WHERE category != ''
			)
			GROUP BY name
		)
		ORDER BY position ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// CreateFolder creates an empty regular folder. The folder becomes visible
// immediately and can later receive feeds via SetFeedCategory.
func (d *DB) CreateFolder(name string) error {
	var nextPos int
	if err := d.sql.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM folders`).Scan(&nextPos); err != nil {
		return fmt.Errorf("next folder position: %w", err)
	}
	_, err := d.sql.Exec(`INSERT OR IGNORE INTO folders (name, position) VALUES (?, ?)`, name, nextPos)
	return err
}

// RenameFolder renames the explicit folder row and all feeds assigned to
// the same category. Empty newName removes the explicit folder and moves
// assigned feeds to Other.
func (d *DB) RenameFolder(oldName, newName string) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if newName == "" {
		if _, err := tx.Exec(`DELETE FROM folders WHERE name = ?`, oldName); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(`UPDATE folders SET name = ? WHERE name = ?`, newName, oldName); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO folders (name, position) VALUES (?, (SELECT COALESCE(MAX(position), -1) + 1 FROM folders))`, newName); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE feeds SET category = ? WHERE category = ?`, newName, oldName); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteFolder removes the explicit folder and moves assigned feeds to Other.
func (d *DB) DeleteFolder(name string) error {
	return d.RenameFolder(name, "")
}
