// Package restic is one of the only two packages that shell out. It selects a
// snapshot and restores the paths a recipe needs.
//
// restic's own environment (RESTIC_REPOSITORY, RESTIC_PASSWORD, RESTIC_PASSWORD_FILE,
// AWS_*, B2_*, ...) is passed through unchanged and is never parsed or logged.
package restic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spelingbee/restored/internal/source"
)

// Options are the repository and snapshot filters for one run.
type Options struct {
	// Repository overrides RESTIC_REPOSITORY. Empty means "use the environment".
	Repository string
	// Snapshot is a snapshot id, an unambiguous prefix of one, or "latest".
	Snapshot string
	Tags     []string
	Host     string
	// Debug receives restic's stderr and the command lines. Never its environment.
	Debug io.Writer
}

func (o Options) baseArgs() []string {
	var args []string
	if o.Repository != "" {
		args = append(args, "--repo", o.Repository)
	}
	return args
}

func (o Options) run(ctx context.Context, args ...string) (string, string, error) {
	full := append(o.baseArgs(), args...)
	cmd := exec.CommandContext(ctx, "restic", full...)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if o.Debug != nil {
		fmt.Fprintf(o.Debug, "+ restic %s\n", strings.Join(full, " "))
	}
	err := cmd.Run()
	if o.Debug != nil && stderr.Len() > 0 {
		fmt.Fprintln(o.Debug, strings.TrimRight(stderr.String(), "\r\n"))
	}
	return stdout.String(), strings.TrimRight(stderr.String(), "\r\n"), err
}

// ListSnapshots returns the snapshots matching the tag and host filters.
func ListSnapshots(ctx context.Context, o Options) ([]source.Snapshot, error) {
	args := []string{"snapshots", "--json"}
	for _, t := range o.Tags {
		args = append(args, "--tag", t)
	}
	if o.Host != "" {
		args = append(args, "--host", o.Host)
	}
	stdout, stderr, err := o.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("restic snapshots: %w: %s", err, lastLines(stderr, 5))
	}
	return ParseSnapshots([]byte(stdout))
}

// ParseSnapshots decodes `restic snapshots --json`. It is separate from the process
// call so snapshot selection can be tested against recorded output.
func ParseSnapshots(raw []byte) ([]source.Snapshot, error) {
	var snaps []source.Snapshot
	if err := json.Unmarshal(bytes.TrimSpace(raw), &snaps); err != nil {
		return nil, fmt.Errorf("parsing restic snapshots --json: %w", err)
	}
	return snaps, nil
}

// Select picks the snapshot a run will use. spec is "latest", empty, a full id, or an
// unambiguous prefix of one.
func Select(snaps []source.Snapshot, spec string) (*source.Snapshot, error) {
	if len(snaps) == 0 {
		return nil, fmt.Errorf("the repository has no snapshots matching the filters")
	}
	if spec == "" || spec == "latest" {
		sorted := make([]source.Snapshot, len(snaps))
		copy(sorted, snaps)
		// Newest first; a tie is broken by id so the choice is deterministic.
		sort.SliceStable(sorted, func(i, j int) bool {
			if !sorted[i].Time.Equal(sorted[j].Time) {
				return sorted[i].Time.After(sorted[j].Time)
			}
			return sorted[i].ID > sorted[j].ID
		})
		out := sorted[0]
		out.SelectedBy = "latest"
		return &out, nil
	}
	spec = strings.ToLower(spec)
	var matches []source.Snapshot
	for _, s := range snaps {
		if strings.HasPrefix(strings.ToLower(s.ID), spec) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no snapshot with id %q in the repository", spec)
	case 1:
		out := matches[0]
		out.SelectedBy = "explicit"
		return &out, nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ShortID)
		}
		return nil, fmt.Errorf("snapshot id %q is ambiguous: %s", spec, strings.Join(ids, ", "))
	}
}

// Restore materialises the requested paths from a snapshot into dest. One restic
// invocation covers every input, so a repository is opened once per run.
func Restore(ctx context.Context, o Options, snap *source.Snapshot, reqs []source.Request, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("creating the restore directory: %w", err)
	}
	args := []string{"restore", snap.ID, "--target", dest}
	seen := map[string]bool{}
	for _, r := range reqs {
		if seen[r.BackupPath] {
			continue
		}
		seen[r.BackupPath] = true
		args = append(args, "--include", r.BackupPath)
	}
	_, stderr, err := o.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("restic restore: %w: %s", err, lastLines(stderr, 8))
	}
	return nil
}

// Locate returns where a backup path landed inside the restore directory.
func Locate(dest, backupPath string) string {
	clean := strings.TrimPrefix(path.Clean(backupPath), "/")
	return filepath.Join(dest, filepath.FromSlash(clean))
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "; ")
}
