package restic

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/spelingbee/drillback/internal/source"
)

// Source is the restic implementation of source.Source. Everything restic-specific
// about a run lives behind it: which snapshot to take, how to restore into the
// workspace, whether the restored tree may be moved out of, and what the report says
// the repository was. See DECISIONS.md ADR-063.
type Source struct {
	Options Options
}

// New returns a restic source for one run.
func New(o Options) *Source { return &Source{Options: o} }

// Kind is the name the --source flag and the report use.
func (s *Source) Kind() string { return "restic" }

// Preflight checks the one thing that has to be true before a workspace is worth
// creating: the binary exists. Whether the repository is readable, and whether the
// password is right, is answered by the first real command rather than guessed at
// here, because guessing costs a second round trip to somebody's remote repository.
func (s *Source) Preflight(_ context.Context) error {
	if _, err := exec.LookPath("restic"); err != nil {
		return fmt.Errorf("restic is not on PATH: --source restic needs the restic binary. " +
			"Install it from https://restic.readthedocs.io, or use --source dir against a " +
			"tree you have already restored")
	}
	return nil
}

// Fetch selects a snapshot and restores every requested path into `into`.
func (s *Source) Fetch(ctx context.Context, reqs []source.Request, into string) (source.Fetched, error) {
	var f source.Fetched

	snaps, err := ListSnapshots(ctx, s.Options)
	if err != nil {
		return f, err
	}
	snap, err := Select(snaps, s.Options.Snapshot)
	if err != nil {
		return f, err
	}
	if err := Restore(ctx, s.Options, snap, reqs, into); err != nil {
		return f, err
	}
	return source.Fetched{
		Descriptor: source.Descriptor{
			Kind:       "restic",
			Repository: SafeRepository(s.Options.RepositoryLabel()),
			Snapshot:   snap,
		},
		Locate: func(backupPath string) string { return Locate(into, backupPath) },
		// The restore directory is the workspace's own and is deleted with the run,
		// so an input can be moved out of it: one copy of the data on disk rather
		// than two, which on a 40 GB Nextcloud is the difference between a drill
		// that fits on the disk and one that does not.
		Disposable: true,
	}, nil
}
