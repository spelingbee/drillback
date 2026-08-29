package harness

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spelingbee/restored/internal/recipe"
	dirsource "github.com/spelingbee/restored/internal/source/dir"
)

// emptyDump is what an empty postgres-dump input contains. psql loads it as a no-op,
// so the database the application meets is the one it creates for itself.
const emptyDump = "-- intentionally empty\n"

// writeEmpty creates the empty shape of one input kind, as SPEC.md section 7.2
// defines it: a dir input is an empty directory, a sqlite input is a zero-length
// file, and a postgres-dump input is a file with nothing but a comment in it.
//
// The shapes matter. A missing file is a restore failure and ends the run before any
// check gets to speak; an empty one is a restore that worked and returned nothing,
// which is exactly the thing stage A has to distinguish a real restore from.
func writeEmpty(kind, p string) error {
	if kind == "dir" {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
	}
	body := ""
	if kind == "postgres-dump" {
		body = emptyDump
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
	}
	return nil
}

// emptyTree builds a tree of empty inputs under root, laid out at the paths the
// recipe says they occupy in a backup. `restored check --source dir --from <root>`
// then walks the real restore path against nothing at all.
func emptyTree(root string, res *recipe.Resolved) error {
	for _, in := range res.Inputs {
		if err := writeEmpty(in.Kind, dirsource.Locate(root, in.BackupPath)); err != nil {
			return err
		}
	}
	return nil
}

// emptyInputs prepares the workspace stage B starts from. It is deliberately NOT
// emptyTree: stage B has no backup to restore, and the application is about to create
// its own world, so the workspace must look like a machine the application has never
// run on rather than like a restore that returned nothing.
//
// The difference matters and cost a session to find. A zero-length kuma.db is a
// perfectly good empty *restore* — SQLite says "file is not a database" and the check
// fails, which is what stage A wants. It is a hopeless empty *start*: Uptime Kuma
// crash-loops rather than running its migrations, and stage B never gets a stack to
// seed. See DECISIONS.md ADR-053.
//
// So stage B creates:
//   - every dir input, as an empty directory;
//   - every non-dir input that compose bind-mounts, as its empty shape, because a
//     bind mount to a path that does not exist makes docker create a directory there.
//
// and creates nothing else: an input that lives inside another one arrives when the
// application writes it, and an input nothing mounts is produced by an export step.
func emptyInputs(res *recipe.Resolved) error {
	for _, in := range res.Inputs {
		switch {
		case in.Within != "":
			continue
		case in.Kind == "dir", in.Mount != nil:
			if err := writeEmpty(in.Kind, in.LocalPath); err != nil {
				return err
			}
		}
	}
	return nil
}
