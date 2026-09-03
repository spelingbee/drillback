package compose

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

// Reader reads the workspace on a check's behalf once the application is running.
//
// It goes through the daemon - root in a throwaway helper container - rather than
// through the calling process, because by then the calling process may no longer be
// able to. An application image that re-owns its mounted data directory on startup
// (FreshRSS chowns to 33:33 and 0770; Trilium and Nextcloud do the equivalent) leaves
// the caller unable to stat the files it restored a moment ago, and `load db` then
// dies with `permission denied` before a single check has run. Windows maps no
// ownership onto bind mounts and never showed the problem, which is how it reached
// CI. Windows takes this path too, so the one path is the one every platform runs.
// See DECISIONS.md ADR-071.
type Reader struct {
	Client      *Client
	HelperImage string
	// InputsDir is the workspace's inputs directory. It is the only host tree the
	// helper sees, and it sees it read-only.
	InputsDir string
	// ReadsDir is where Fetch leaves its copies. It belongs to the caller, so the
	// caller can open what root put there and remove it afterwards.
	ReadsDir string

	seq atomic.Int64
}

// Entry is one path as the daemon saw it.
type Entry struct {
	// Rel is slash-separated and relative to the listed path; "" is the path itself.
	Rel   string
	IsDir bool
	Size  int64
}

const (
	inputsMount = "/inputs"
	readsMount  = "/reads"
	// exitMissing is the helper's answer when there is nothing at the path. That is
	// an observation - `exists: false` is a thing a check may expect - not a failure.
	exitMissing = 3
	// exitNotAFile is the helper's answer when Fetch is pointed at a directory.
	exitNotAFile = 4
)

// listScript prints one line per path: type, size, and the name as find saw it. The
// name comes last because it may contain the separator. Errors from the walk itself
// are dropped on purpose: the tree belongs to a running application, and a temporary
// file that vanished between find and stat is not a finding about the backup.
const listScript = `[ -e "$1" ] || exit 3
find "$1" -maxdepth "$2" -exec stat -c '%F|%s|%n' {} + 2>/dev/null
exit 0`

// fetchScript copies one regular file, plus the -wal and -shm sidecars SQLite may have
// put beside it, and opens the copies' modes so the caller can read them.
const fetchScript = `set -e
[ -e "$1" ] || exit 3
[ -f "$1" ] || exit 4
cp "$1" "$2/"
for s in -wal -shm; do if [ -e "$1$s" ]; then cp "$1$s" "$2/"; fi; done
chmod a+rw "$2"/*`

// List returns hostPath and everything under it to depth levels below it. exists
// reports whether there was anything at hostPath at all; when there was not, entries
// is nil and err is nil.
func (r *Reader) List(ctx context.Context, hostPath string, depth int) (entries []Entry, exists bool, err error) {
	inside, err := r.containerPath(hostPath)
	if err != nil {
		return nil, false, err
	}
	res, err := r.run(ctx, listScript, []string{inside, strconv.Itoa(depth)}, nil)
	if err != nil {
		return nil, false, fmt.Errorf("listing %s: %w", hostPath, err)
	}
	switch res.ExitCode {
	case 0:
	case exitMissing:
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("listing %s: helper exited %d: %s",
			hostPath, res.ExitCode, firstLines(res.Combined(), 4))
	}
	entries = parseListing(res.Stdout, inside)
	if !hasSelf(entries) {
		return nil, false, fmt.Errorf("listing %s: the helper printed nothing for the path itself", hostPath)
	}
	return entries, true, nil
}

// Fetch copies the regular file at hostPath into a fresh directory under ReadsDir and
// returns the copy's path with a function that removes it. A SQLite database's -wal
// and -shm sidecars travel with it, so the copy reads as the original would have.
//
// A missing file is reported as fs.ErrNotExist, which is what an empty stack in the
// harness's stage A produces and what its verdict is keyed on.
func (r *Reader) Fetch(ctx context.Context, hostPath string) (string, func(), error) {
	inside, err := r.containerPath(hostPath)
	if err != nil {
		return "", nil, err
	}
	name := strconv.FormatInt(r.seq.Add(1), 10)
	dir := filepath.Join(r.ReadsDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	res, err := r.run(ctx, fetchScript, []string{inside, path.Join(readsMount, name)},
		&Bind{Host: filepath.ToSlash(r.ReadsDir), Container: readsMount})
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("copying %s out of the workspace: %w", filepath.Base(hostPath), err)
	}
	switch res.ExitCode {
	case 0:
	case exitMissing:
		cleanup()
		return "", nil, fs.ErrNotExist
	case exitNotAFile:
		cleanup()
		return "", nil, fmt.Errorf("%s is not a regular file", filepath.Base(hostPath))
	default:
		cleanup()
		return "", nil, fmt.Errorf("copying %s out of the workspace: helper exited %d: %s",
			filepath.Base(hostPath), res.ExitCode, firstLines(res.Combined(), 4))
	}
	return filepath.Join(dir, filepath.Base(hostPath)), cleanup, nil
}

// run executes a shell script as root in the helper, with the inputs tree bound
// read-only and, when asked, one more bind.
func (r *Reader) run(ctx context.Context, script string, args []string, extra *Bind) (Result, error) {
	entrypoint := ""
	binds := []Bind{{Host: filepath.ToSlash(r.InputsDir), Container: inputsMount, ReadOnly: true}}
	if extra != nil {
		binds = append(binds, *extra)
	}
	argv := append([]string{"sh", "-c", script, "sh"}, args...)
	return r.Client.RunContainer(ctx, ContainerOptions{
		Image: r.HelperImage,
		// The helper image declares an unprivileged USER, which is right for probes
		// and useless here: the point is root's authority over another uid's modes.
		User:       "0:0",
		Entrypoint: &entrypoint,
		Binds:      binds,
		Argv:       argv,
	})
}

// containerPath maps a workspace path to where the helper sees it. Anything outside
// the inputs tree is refused: the helper is bound to that tree and nothing else.
func (r *Reader) containerPath(hostPath string) (string, error) {
	rel, err := filepath.Rel(r.InputsDir, hostPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to read %q: it is not under the workspace's inputs", hostPath)
	}
	return path.Join(inputsMount, filepath.ToSlash(rel)), nil
}

// parseListing turns listScript's output into entries relative to inside. Lines that
// are not three fields, or that name something outside inside, are skipped.
func parseListing(out, inside string) []Entry {
	var entries []Entry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		kind, rest, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		sizeText, name, ok := strings.Cut(rest, "|")
		if !ok {
			continue
		}
		if name != inside && !strings.HasPrefix(name, inside+"/") {
			continue
		}
		size, _ := strconv.ParseInt(sizeText, 10, 64)
		rel := strings.TrimPrefix(strings.TrimPrefix(name, inside), "/")
		entries = append(entries, Entry{Rel: rel, IsDir: kind == "directory", Size: size})
	}
	return entries
}

func hasSelf(entries []Entry) bool {
	for _, e := range entries {
		if e.Rel == "" {
			return true
		}
	}
	return false
}
