// Package check runs one resolved check and evaluates what it saw. It does not know
// what a recipe is beyond the check struct it is handed, which is what lets the same
// code serve the ready probes. See SPEC.md section 13.1.
package check

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spelingbee/drillback/internal/compose"
	"github.com/spelingbee/drillback/internal/recipe"
	"github.com/spelingbee/drillback/internal/sqlite"
)

// statusMarker separates the response body from the status code curl appends. It is
// long and unlikely enough that a body containing it would already be pathological,
// and only the last occurrence is read.
const statusMarker = "\n###drillback-http-status:"

// Mount is one place where a container path and a workspace path are the same bytes.
// It is how a `file` check looks at what a service sees without entering a container.
type Mount struct {
	Service       string
	ContainerPath string
	HostPath      string
}

// Executor holds everything a check needs in order to run, and nothing about recipes.
type Executor struct {
	Compose     *compose.Client
	Network     string
	HelperImage string
	Mounts      []Mount
	// Reader is how a check reads the workspace once the application is running,
	// which may by then own what it was handed. See compose.Reader.
	Reader *compose.Reader
}

// Result is one check, run and judged.
type Result struct {
	Check    *recipe.Check
	Status   string
	Duration time.Duration
	Observed Observation
	Failures []Failure
}

// Passed reports the verdict for one check.
func (r Result) Passed() bool { return r.Status == "pass" }

// Run executes one check and evaluates it. It never returns an error: a check that
// could not run is a failing check with the reason in observed.error, because the
// report has to show every check either way.
func Run(ctx context.Context, e *Executor, c *recipe.Check, timeout time.Duration) Result {
	if c.Timeout != "" {
		if d, err := time.ParseDuration(c.Timeout); err == nil {
			timeout = d
		}
	}
	start := time.Now()
	var obs Observation
	switch c.Kind {
	case "http":
		obs = e.HTTP(ctx, HTTPRequest{
			URL: c.URL, Method: c.Method, BasicAuth: c.BasicAuth,
			JSONBody: c.JSONBody, Timeout: timeout,
		})
	case "exec":
		obs = e.Exec(ctx, c.Service, c.User, c.Command, timeout)
	case "sql":
		obs = e.SQL(ctx, c, timeout)
	case "file":
		obs = e.File(ctx, c)
	default:
		obs = Observation{Error: fmt.Sprintf("unknown check kind %q", c.Kind)}
	}
	res := Result{Check: c, Duration: time.Since(start), Observed: obs}
	res.Failures = Evaluate(c.Kind, c.Expect, &res.Observed)
	if len(res.Failures) == 0 {
		res.Status = "pass"
	} else {
		res.Status = "fail"
	}
	return res
}

// HTTPRequest is one request made from a helper container on the run's network.
type HTTPRequest struct {
	URL       string
	Method    string
	BasicAuth []string
	JSONBody  string
	Timeout   time.Duration
}

// HTTP makes a request from a throwaway container attached to the run's internal
// network. drillback publishes no ports, so this is the only way in.
func (e *Executor) HTTP(ctx context.Context, r HTTPRequest) Observation {
	secs := int(r.Timeout.Seconds())
	if secs < 1 {
		secs = 60
	}
	argv := []string{"curl", "-sS", "-o", "-",
		"-w", statusMarker + "%{http_code}",
		"--max-time", strconv.Itoa(secs)}
	switch strings.ToUpper(r.Method) {
	case "", "GET":
	case "HEAD":
		argv = append(argv, "-I")
	default:
		argv = append(argv, "-X", strings.ToUpper(r.Method))
	}
	if len(r.BasicAuth) == 2 {
		argv = append(argv, "-u", r.BasicAuth[0]+":"+r.BasicAuth[1])
	}
	if r.JSONBody != "" {
		argv = append(argv, "-H", "Content-Type: application/json", "--data-binary", r.JSONBody)
	}
	argv = append(argv, r.URL)

	runCtx, cancel := context.WithTimeout(ctx, r.Timeout+15*time.Second)
	defer cancel()
	res, err := e.Compose.RunHelper(runCtx, compose.RunOptions{
		Image: e.HelperImage, Network: e.Network, Argv: argv,
	})
	if err != nil {
		return Observation{Error: err.Error()}
	}

	idx := strings.LastIndex(res.Stdout, statusMarker)
	if idx < 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = "no response from " + r.URL
		}
		return Observation{Error: msg}
	}
	body := res.Stdout[:idx]
	code, convErr := strconv.Atoi(strings.TrimSpace(res.Stdout[idx+len(statusMarker):]))
	if convErr != nil || code == 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = "the request did not complete"
		}
		return Observation{Error: msg}
	}
	n := len(body)
	return Observation{Status: &code, BodyBytes: &n, Body: body}
}

// Exec runs a command inside a service. The argv is passed through unchanged.
func (e *Executor) Exec(ctx context.Context, service, user string, argv []string, timeout time.Duration) Observation {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := e.Compose.Exec(runCtx, compose.ExecOptions{Service: service, User: user, Argv: argv})
	if err != nil {
		return Observation{Error: err.Error()}
	}
	code := res.ExitCode
	return Observation{ExitCode: &code, Stdout: res.Stdout, Stderr: res.Stderr}
}

// TCP reports whether a service is accepting connections on a port, from a helper
// container on the run's network.
func (e *Executor) TCP(ctx context.Context, service string, port int, timeout time.Duration) Observation {
	runCtx, cancel := context.WithTimeout(ctx, timeout+15*time.Second)
	defer cancel()
	res, err := e.Compose.RunHelper(runCtx, compose.RunOptions{
		Image:   e.HelperImage,
		Network: e.Network,
		Argv:    []string{"nc", "-z", "-w", "3", service, strconv.Itoa(port)},
	})
	if err != nil {
		return Observation{Error: err.Error()}
	}
	code := res.ExitCode
	obs := Observation{ExitCode: &code, Stdout: res.Stdout, Stderr: res.Stderr}
	if code != 0 {
		obs.Error = fmt.Sprintf("%s:%d is not accepting connections", service, port)
	}
	return obs
}

// SQL runs a query against PostgreSQL inside its service, or against a restored
// SQLite file in the workspace.
func (e *Executor) SQL(ctx context.Context, c *recipe.Check, timeout time.Duration) Observation {
	switch c.Driver {
	case "sqlite":
		return e.sqliteQuery(ctx, c.File, c.Query, timeout)
	case "postgres":
		return e.postgresQuery(ctx, c, timeout)
	default:
		return Observation{Error: fmt.Sprintf("unknown sql driver %q", c.Driver)}
	}
}

// sqliteQuery opens a copy the daemon made, not the restored file itself: the
// application that has just started may own that file by now. See compose.Reader.
func (e *Executor) sqliteQuery(ctx context.Context, file, query string, timeout time.Duration) Observation {
	if e.Reader == nil {
		return Observation{Error: "no workspace reader is configured for this run"}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	local, done, err := e.Reader.Fetch(runCtx, file)
	if err != nil {
		return Observation{Error: fmt.Sprintf("opening %s: %v", filepath.Base(file), err)}
	}
	defer done()
	rows, err := sqlite.Query(runCtx, local, query)
	if err != nil {
		return Observation{Error: err.Error()}
	}
	return rowsObservation(rows)
}

// postgresQuery runs psql inside the database service. Unaligned, tuples-only output
// is the closest psql gets to a machine-readable result without adding a driver and a
// published port to reach it through.
func (e *Executor) postgresQuery(ctx context.Context, c *recipe.Check, timeout time.Duration) Observation {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := e.Compose.Exec(runCtx, compose.ExecOptions{
		Service: c.Service,
		Argv: []string{"psql", "-v", "ON_ERROR_STOP=1", "-t", "-A", "-F", "\x1f",
			"-U", c.User, "-d", c.Database, "-c", c.Query},
	})
	if err != nil {
		return Observation{Error: err.Error()}
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		return Observation{Error: msg}
	}
	var rows [][]string
	for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\r\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\x1f"))
	}
	return rowsObservation(rows)
}

func rowsObservation(rows [][]string) Observation {
	n := len(rows)
	obs := Observation{Rows: &n}
	if n > 0 && len(rows[0]) > 0 {
		obs.Value = strings.TrimSpace(rows[0][0])
		obs.Summary = obs.Value
	}
	return obs
}

// File inspects a path a service sees, through the workspace rather than through the
// container, so a check costs no process in the application's image. The workspace is
// listed by the daemon rather than by this process, because the application that has
// just started may own that path by now. See compose.Reader.
func (e *Executor) File(ctx context.Context, c *recipe.Check) Observation {
	if e.Reader == nil {
		return Observation{Error: "no workspace reader is configured for this run"}
	}
	host, err := e.HostPath(c.Service, c.Path)
	if err != nil {
		return Observation{Error: err.Error()}
	}
	entries, exists, err := e.Reader.List(ctx, host, globDepth(c.Expect.Glob))
	if err != nil {
		return Observation{Error: err.Error()}
	}
	return observeTree(entries, exists, c.Expect.Glob)
}

// globDepth is how far below the path a listing has to reach: one level for the
// entries `not_empty` counts, and one per segment for a glob to be judged.
func globDepth(glob string) int {
	glob = strings.Trim(glob, "/")
	if glob == "" {
		return 1
	}
	return strings.Count(glob, "/") + 1
}

// observeTree judges a listing the way the host used to judge the tree itself: the
// path's own type and size, the number of direct entries under a directory, and the
// number of matches for a glob. path.Match is filepath.Glob's own matcher, so a
// pattern means what it always meant.
func observeTree(entries []compose.Entry, exists bool, glob string) Observation {
	var obs Observation
	obs.Exists = &exists
	if !exists {
		return obs
	}
	var self *compose.Entry
	for i := range entries {
		if entries[i].Rel == "" {
			self = &entries[i]
			break
		}
	}
	if self == nil {
		obs.Error = "the listing did not include the path itself"
		return obs
	}
	isDir := self.IsDir
	obs.IsDir = &isDir
	if !isDir {
		size := self.Size
		obs.Bytes = &size
		obs.Summary = fmt.Sprintf("%d bytes", size)
	} else {
		n := 0
		for _, en := range entries {
			if en.Rel != "" && !strings.Contains(en.Rel, "/") {
				n++
			}
		}
		obs.Entries = &n
	}
	if glob != "" {
		pattern := strings.Trim(glob, "/")
		n := 0
		for _, en := range entries {
			if en.Rel == "" {
				continue
			}
			ok, matchErr := path.Match(pattern, en.Rel)
			if matchErr != nil {
				obs.Error = fmt.Sprintf("glob %q: %v", glob, matchErr)
				return obs
			}
			if ok {
				n++
			}
		}
		obs.Count = &n
		obs.Summary = fmt.Sprintf("%d match%s for %s", n, matchPlural(n), glob)
	}
	return obs
}

// HostPath maps a path inside a service to the workspace path holding the same bytes.
func (e *Executor) HostPath(service, p string) (string, error) {
	clean := path.Clean(p)
	best := Mount{}
	for _, m := range e.Mounts {
		if m.Service != service {
			continue
		}
		cp := path.Clean(m.ContainerPath)
		if clean != cp && !strings.HasPrefix(clean, cp+"/") {
			continue
		}
		if len(cp) > len(path.Clean(best.ContainerPath)) || best.HostPath == "" {
			best = m
		}
	}
	if best.HostPath == "" {
		return "", fmt.Errorf("no input of this recipe is mounted at %s in service %q, "+
			"so drillback cannot see that path", p, service)
	}
	rel := strings.TrimPrefix(clean, path.Clean(best.ContainerPath))
	return filepath.Join(best.HostPath, filepath.FromSlash(strings.TrimPrefix(rel, "/"))), nil
}

func matchPlural(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}
