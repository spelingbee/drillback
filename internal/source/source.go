// Package source holds the types every backup source shares. The sources themselves
// live in the subpackages: restic and dir.
package source

import (
	"context"
	"time"
)

// Snapshot is one point in time in a backup repository.
type Snapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Tags     []string  `json:"tags"`
	Paths    []string  `json:"paths"`

	// SelectedBy records how this snapshot was chosen: "latest" or "explicit".
	SelectedBy string `json:"-"`
}

// Request is one input the run needs materialised.
type Request struct {
	Name       string
	BackupPath string
	LocalPath  string
	Required   bool
}

// Descriptor is what the report says about where the data came from.
type Descriptor struct {
	Kind       string    `json:"kind"`
	Repository string    `json:"repository,omitempty"`
	Snapshot   *Snapshot `json:"snapshot,omitempty"`
}

// Source is a backup a run can read.
//
// ADR-004 recorded, as a consequence, that "borg and kopia land in v0.2 behind the
// same `source` interface, which is why that interface exists in v0.1 with only two
// implementations". It did not: this package held three structs and no method, restic
// was called through free functions from a `switch` in the runner, and restic-shaped
// decisions leaked past that switch into the lifecycle - which is to say, adding borg
// meant editing five packages. The session 4 architecture review found the gap
// (docs/review/architecture.md ARCH-02) and this is the interface being made real
// before a third source rather than with it. See DECISIONS.md ADR-063.
type Source interface {
	// Kind is the name the report and the --source flag use.
	Kind() string

	// Preflight reports whether this source can be read at all: its binary is
	// installed, its repository exists, its arguments make sense. It runs before the
	// workspace is created, so it must not need one.
	Preflight(ctx context.Context) error

	// Fetch brings every request into the directory `into`, which belongs to the run.
	// A source that already has the data on disk may ignore `into` entirely.
	Fetch(ctx context.Context, reqs []Request, into string) (Fetched, error)
}

// Fetched is what a Fetch produced.
type Fetched struct {
	// Descriptor is what the report says about where the data came from.
	Descriptor Descriptor

	// Locate maps a path as it appears in the backup to a path on this machine.
	Locate func(backupPath string) string

	// Disposable says the tree Locate points into belongs to this run and will be
	// deleted with it, so the runner may move an input out of it rather than copy it.
	// It is false for a source that points at something the user owns - moving files
	// out of somebody's live directory is not a thing a restore drill may do.
	Disposable bool
}
