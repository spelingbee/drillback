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
			resolved = filepath.Join(filepath.Dir(p), target)
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
