package restic

import (
	"path/filepath"

	"github.com/spelingbee/restored/internal/source"
)

// snapshotAlias keeps the test readable without importing internal/source in every
// assertion.
type snapshotAlias = source.Snapshot

// filepathSlash normalises a path for comparison, so the assertions read the same on
// Windows as on Linux.
func filepathSlash(p string) string { return filepath.ToSlash(p) }
