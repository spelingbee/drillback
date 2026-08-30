// Package sqlite reads a restored SQLite database from the workspace.
//
// The database is opened in this process rather than inside a container, because the
// file is a workspace path and because it means a recipe does not have to ship a
// sqlite3 binary in the application's image to be checkable. See DECISIONS.md ADR-040.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // the pure-Go driver, registered as "sqlite"
)

// Query runs one statement and returns every row as strings, which is the only shape
// the expect vocabulary needs.
func Query(ctx context.Context, file, query string) ([][]string, error) {
	if _, err := os.Stat(file); err != nil {
		return nil, fmt.Errorf("opening %s: %w", filepath.Base(file), err)
	}
	db, err := sql.Open("sqlite", dsn(file))
	if err != nil {
		return nil, fmt.Errorf("opening the database: %w", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out [][]string
	for rows.Next() {
		cells := make([]any, len(cols))
		for i := range cells {
			cells[i] = new(sql.NullString)
		}
		if err := rows.Scan(cells...); err != nil {
			return nil, err
		}
		row := make([]string, len(cols))
		for i, c := range cells {
			if ns := c.(*sql.NullString); ns.Valid {
				row[i] = ns.String
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// dsn opens the file read-only. drillback never writes to a restored database: a check
// that mutates the thing it is checking is not a check.
func dsn(file string) string {
	u := url.URL{Scheme: "file", Opaque: filepath.ToSlash(file)}
	q := url.Values{}
	q.Set("mode", "ro")
	q.Set("_pragma", "busy_timeout(5000)")
	return u.String() + "?" + q.Encode()
}
