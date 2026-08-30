// Package dir materialises inputs from a tree that is already restored on disk. It is
// the source that needs no backup repository, and it is what makes the first-run
// experience cost nothing.
package dir

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Locate returns where a backup path lives inside an already-restored tree. The tree
// mirrors the filesystem the backup was taken from, so /srv/gitea/data inside the
// backup is <from>/srv/gitea/data on disk.
func Locate(from, backupPath string) string {
	clean := strings.TrimPrefix(path.Clean(backupPath), "/")
	return filepath.Join(from, filepath.FromSlash(clean))
}

// Check reports whether the tree exists and is a directory, with the error a user can
// act on rather than a bare ENOENT.
func Check(from string) error {
	info, err := os.Stat(from)
	if err != nil {
		return fmt.Errorf("--from %q: %w", from, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("--from %q is not a directory: with --source dir it is the "+
			"root of an already-restored tree", from)
	}
	return nil
}

// errNoTree is the message for `--source dir` with no tree to read.
var errNoTree = fmt.Errorf("--source dir needs --from <tree>: the root of a tree you " +
	"have already restored, which restored reads instead of running a restore itself")
