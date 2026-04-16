package db

// SmartFolder is a saved query that appears as a virtual folder in the
// feed list. Selecting it loads articles from across all feeds matching
// Query (using the rdr query language; see internal/ui/query.go).
type SmartFolder struct {
	ID       int64
	Name     string
	Query    string
	Position int
}

// ListSmartFolders returns all smart folders ordered by position then id
// so the UI draws them in a stable, user-controlled sequence.
func (d *DB) ListSmartFolders() ([]SmartFolder, error) {
	rows, err := d.sql.Query(`
		SELECT id, name, query, position
		FROM smart_folders
		ORDER BY position, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SmartFolder
	for rows.Next() {
		var f SmartFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Query, &f.Position); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// InsertSmartFolder appends a new folder at the end of the list. Position
// is computed from the current max so new folders land after existing ones.
func (d *DB) InsertSmartFolder(name, query string) (SmartFolder, error) {
	res, err := d.sql.Exec(`
		INSERT INTO smart_folders (name, query, position)
		VALUES (?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM smart_folders))
	`, name, query)
	if err != nil {
		return SmartFolder{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SmartFolder{}, err
	}
	var f SmartFolder
	err = d.sql.QueryRow(`
		SELECT id, name, query, position
		FROM smart_folders
		WHERE id = ?
	`, id).Scan(&f.ID, &f.Name, &f.Query, &f.Position)
	return f, err
}

// UpdateSmartFolder overwrites the name and query of an existing folder.
// Position is preserved so the user's ordering survives edits.
func (d *DB) UpdateSmartFolder(id int64, name, query string) error {
	_, err := d.sql.Exec(`
		UPDATE smart_folders SET name = ?, query = ? WHERE id = ?
	`, name, query, id)
	return err
}

// DeleteSmartFolder removes a folder by id. No-op if the id does not
// exist so callers don't have to pre-check.
func (d *DB) DeleteSmartFolder(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM smart_folders WHERE id = ?`, id)
	return err
}
