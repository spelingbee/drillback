// Package workspace owns every path a run touches.
//
// No other package builds a path inside the run directory; they ask for one. That is
// what makes "nothing outside the workspace" a structural property of the program
// rather than a habit. See SPEC.md section 13.1.
package workspace

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Workspace is one run's directory tree.
type Workspace struct {
	RunID string
	Root  string
}

// New allocates a run id and creates <parent>/drillback-<runid> with its subdirectories.
// parent may be empty, in which case the OS temp directory is used.
func New(parent string) (*Workspace, error) {
	id, err := NewRunID()
	if err != nil {
		return nil, err
	}
	return NewWithID(parent, id)
}

// NewWithID is New with the run id supplied, which the tests rely on.
func NewWithID(parent, id string) (*Workspace, error) {
	if parent == "" {
		parent = os.TempDir()
	}
	abs, err := filepath.Abs(parent)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace parent %q: %w", parent, err)
	}
	root := filepath.Join(abs, "drillback-"+id)
	// 0700, not 0755. The workspace holds a restored copy of somebody's backup, in a
	// directory that is usually world-traversable, and the harness deliberately makes
	// the input trees inside it world-writable so that a container running as its own
	// uid can start (DECISIONS.md ADR-054). Both of those are only safe because
	// nobody else can get through this directory to reach them.
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("creating workspace %q: %w", root, err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("securing workspace %q: %w", root, err)
	}
	ws := &Workspace{RunID: id, Root: root}
	for _, d := range []string{ws.InputsDir(), ws.RestoreDir(), ws.LogsDir(),
		ws.TestAssetsDir(), ws.ExportDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("creating %q: %w", d, err)
		}
	}
	return ws, nil
}

// NewRunID returns the short, lower-case, collision-resistant identifier that names
// the workspace, the compose project, and every label drillback sets.
func NewRunID() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("allocating a run id: %w", err)
	}
	enc := base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)
	return enc.EncodeToString(b), nil
}

// InputsDir holds one entry per resolved input.
func (w *Workspace) InputsDir() string { return filepath.Join(w.Root, "inputs") }

// RestoreDir holds the raw output of the backup source before inputs are extracted.
func (w *Workspace) RestoreDir() string { return filepath.Join(w.Root, "restore") }

// LogsDir holds captured service logs and the debug log.
func (w *Workspace) LogsDir() string { return filepath.Join(w.Root, "logs") }

// TestAssetsDir backs ${DRILLBACK_TEST_ASSETS}. During `check` it is empty, because no
// harness service starts; it exists so that compose can interpolate a recipe that
// declares one without reaching outside the workspace.
func (w *Workspace) TestAssetsDir() string { return filepath.Join(w.Root, "test-assets") }

// ExportDir backs ${DRILLBACK_EXPORT}, where the harness collects what it exported.
func (w *Workspace) ExportDir() string { return filepath.Join(w.Root, "export") }

// ComposeFile is the interpolated compose file drillback writes and runs.
func (w *Workspace) ComposeFile() string { return filepath.Join(w.Root, "compose.yaml") }

// ProjectName is the compose project for this run.
func (w *Workspace) ProjectName() string { return "drillback-" + w.RunID }

// Remove deletes the whole workspace. It is idempotent.
//
// The retry is for Windows, where a file another process has only just stopped using
// can stay locked for a moment after the process that held it exited.
func (w *Workspace) Remove() error {
	if w.Root == "" {
		return nil
	}
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		if err = os.RemoveAll(w.Root); err == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	return fmt.Errorf("removing workspace %q: %w", w.Root, err)
}

// Contains reports whether p is inside the workspace. Every path that reaches a
// container goes through this.
func (w *Workspace) Contains(p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(w.Root, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Stat is the size and file count of one materialised input, so an empty restore is
// visible in the report even when every check somehow passes.
type Stat struct {
	Bytes int64
	Files int
}

// Measure walks a path and totals it.
func Measure(p string) (Stat, error) {
	var s Stat
	info, err := os.Lstat(p)
	if err != nil {
		return s, err
	}
	if !info.IsDir() {
		return Stat{Bytes: info.Size(), Files: 1}, nil
	}
	err = filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		s.Files++
		s.Bytes += fi.Size()
		return nil
	})
	return s, err
}

// CopyTree copies src to dst, following no symlinks and creating no links of its own.
// It is how the dir source materialises an input without touching the user's tree.
func CopyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst, info.Mode())
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&os.ModeSymlink != 0:
			// A symlink from a user-supplied tree is neutralised the same way a
			// restored one is; see Sanitise.
			return os.WriteFile(target, nil, 0o644)
		default:
			fi, err := d.Info()
			if err != nil {
				return err
			}
			return copyFile(p, target, fi.Mode())
		}
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
