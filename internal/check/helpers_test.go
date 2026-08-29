package check

import "path/filepath"

// slash normalises a path so the assertions read the same on Windows as on Linux.
func slash(p string) string { return filepath.ToSlash(p) }
