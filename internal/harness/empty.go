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
// Empty inputs are created world-writable, and that is deliberate. An application in
// a container runs as whatever uid its image chose - Gitea is 1000, Nextcloud is 33,
// Paperless is 1000 - and none of them is the uid running restored. On Linux a bind
// mount carries the host's permissions straight through, so a 0755 directory owned by
// the caller is a directory the application cannot write, and stage B never gets a
// stack that starts. On Windows the mode is ignored.
//
// The exposure this buys is bounded by the workspace, which is created 0700 by
// internal/workspace: no other local user can traverse into it to reach these files,
// and the whole tree is destroyed when the run ends. See DECISIONS.md ADR-054.
const (
	emptyDirMode  = 0o777
	emptyFileMode = 0o666
)

func writeEmpty(kind, p string) error {
	if kind == "dir" {
		if err := os.MkdirAll(p, emptyDirMode); err != nil {
			return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
		}
		// MkdirAll applies the process umask, which is 022 on most machines and
		// would quietly turn 0777 into 0755. Chmod does not.
		if err := os.Chmod(p, emptyDirMode); err != nil {
			return fmt.Errorf("opening the empty %s input at %s: %w", kind, p, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), emptyDirMode); err != nil {
		return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
	}
	_ = os.Chmod(filepath.Dir(p), emptyDirMode)
	body := ""
	if kind == "postgres-dump" {
		body = emptyDump
	}
	if err := os.WriteFile(p, []byte(body), emptyFileMode); err != nil {
		return fmt.Errorf("creating the empty %s input at %s: %w", kind, p, err)
	}
	if err := os.Chmod(p, emptyFileMode); err != nil {
		return fmt.Errorf("opening the empty %s input at %s: %w", kind, p, err)
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
