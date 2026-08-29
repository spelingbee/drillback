// Package source holds the types every backup source shares. The sources themselves
// live in the subpackages: restic and dir.
package source

import "time"

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
