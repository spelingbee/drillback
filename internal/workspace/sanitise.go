package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Warning is one thing the sanitisation pass changed or refused, for the report.
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Modes Relax gives a restored tree. See its own comment for why.
const (
	relaxDirMode  = 0o777
	relaxFileMode = 0o666
)

// Relax makes a restored tree usable by whatever uid the application's image runs as.
//
// A backup preserves the ownership of the machine it was taken from, and restic
// restores it faithfully. That ownership means nothing here: the application is about
// to start in a fresh container as the uid its image chose - 33 for Nextcloud, 1000
// for Gitea and Paperless - and a tree owned by uid 1001 with mode 0770 is a tree that
// application cannot read. Nextcloud answers 503 "your data directory is not
// writable"; Gitea logs a permission error and exits. Neither is a fact about the
// backup, and reporting either as an unusable restore would be a false alarm of
// exactly the kind this tool exists to remove.
//
// So the modes are opened up and the ownership is left alone. It is safe because the
// workspace root is 0700 and lives for the length of one run: no other user on the
// machine can traverse into it, and it is deleted on every exit path. See
// DECISIONS.md ADR-055 and SPEC.md section 4.3.
func (w *Workspace) Relax(root string) error {
	if !w.Contains(root) {
		return fmt.Errorf("refusing to open up %q: it is outside the workspace", root)
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil // Sanitise has already dealt with these
		}
		mode := fs.FileMode(relaxFileMode)
		if d.IsDir() {
			mode = relaxDirMode
		}
		// A file restored read-only is the common case, and failing the whole run
		// over one of them would be worse than the application meeting it.
		_ = os.Chmod(p, mode)
		return nil
	})
}

// Sanitise walks a materialised input once, before anything is mounted, and makes it
// safe to hand to a container. See SPEC.md section 4.3.
//
//   - a symlink whose target escapes the workspace is replaced with a zero-byte file
//     (a backup containing an /etc/shadow symlink is a real thing, and it must not
//     become a read of the host's /etc/shadow from inside a container);
//   - a path component equal to ".." is refused outright.
func (w *Workspace) Sanitise(root string) ([]Warning, error) {
	if !w.Contains(root) {
		return nil, fmt.Errorf("refusing to sanitise %q: it is outside the workspace", root)
	}
	var warnings []Warning
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(w.Root, p)
		if relErr != nil {
			return relErr
		}
		for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
			if seg == ".." {
				return fmt.Errorf("restored tree contains a %q path component: %s", "..", rel)
			}
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		target, readErr := os.Readlink(p)
		if readErr != nil {
			return readErr
		}
		resolved := target
		if !filepath.IsAbs(resolved) {
			if resolved != "" && (resolved[0] == '/' || resolved[0] == '\\') {
				// Rooted but driveless - "\etc\shadow". Windows's filepath.IsAbs
				// says false for it, and filepath.Join would glue it under the
				// link's own directory, so an escaping link passed containment
				// and stayed alive. Resolve it the way the OS does, against the
				// volume the workspace lives on, and let Contains judge that.
				// First caught by the first CI run's windows-latest job: every
				// developer host before it either skipped the test or was POSIX.
				resolved = filepath.VolumeName(w.Root) + filepath.FromSlash(resolved)
			} else {
				resolved = filepath.Join(filepath.Dir(p), target)
			}
		}
		if w.Contains(resolved) {
			return nil
		}
		if rmErr := os.Remove(p); rmErr != nil {
			return rmErr
		}
		if wErr := os.WriteFile(p, nil, 0o644); wErr != nil {
			return wErr
		}
		warnings = append(warnings, Warning{
			Code:   "symlink_escaped_workspace",
			Detail: fmt.Sprintf("%s -> %s (neutralised)", filepath.ToSlash(rel), target),
		})
		return nil
	})
	if err != nil {
		return warnings, err
	}
	return warnings, nil
}
