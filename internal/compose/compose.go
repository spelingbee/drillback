// Package compose is one of the only two packages that shell out. It invokes
// docker compose v2 for one run, names the project, labels everything it creates, and
// tears it all down again. See SPEC.md section 13.1.
package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// LabelRun is set on every object drillback creates, so orphans from a crashed process
// are always findable with `docker ps -aq --filter label=com.drillback.run`.
const LabelRun = "com.drillback.run"

// Client drives one compose project.
type Client struct {
	Project string
	File    string
	RunID   string
	// Profiles are the compose profiles this client activates. `drillback check`
	// activates none; the round-trip harness activates "test", which is how a
	// recipe's seeder service exists during `recipe test` and nowhere else.
	Profiles []string
	// Env is passed to every docker invocation on top of the process environment.
	Env map[string]string
	// Debug receives the command lines and the child processes' stderr. It never
	// receives environment values, because those carry secrets.
	Debug io.Writer
}

// Result is the outcome of one command. A non-zero ExitCode is data, not an error:
// checks need to see it.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Combined is stdout and stderr in the order a human would want to read them.
func (r Result) Combined() string {
	switch {
	case r.Stdout == "":
		return r.Stderr
	case r.Stderr == "":
		return r.Stdout
	default:
		return r.Stdout + "\n" + r.Stderr
	}
}

func (c *Client) run(ctx context.Context, stdin io.Reader, args ...string) (Result, error) {
	return c.runEnv(ctx, nil, stdin, args...)
}

// runEnv is run with extra environment for this invocation only. Values never reach
// the debug log: the argument list does, and the environment deliberately does not,
// because that is where a repository password lives.
func (c *Client) runEnv(ctx context.Context, extra map[string]string, stdin io.Reader, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ()
	for k, v := range c.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range extra {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin

	c.debugf("+ docker %s", strings.Join(args, " "))
	err := cmd.Run()
	res := Result{
		Stdout: strings.TrimRight(stdout.String(), "\r\n"),
		Stderr: strings.TrimRight(stderr.String(), "\r\n"),
	}
	if res.Stderr != "" {
		c.debugf("%s", res.Stderr)
	}
	var ee *exec.ExitError
	switch {
	case err == nil:
		return res, nil
	case errors.As(err, &ee):
		res.ExitCode = ee.ExitCode()
		return res, nil
	default:
		return res, fmt.Errorf("running docker %s: %w", args[0], err)
	}
}

func (c *Client) debugf(format string, args ...any) {
	if c.Debug == nil {
		return
	}
	_, _ = fmt.Fprintf(c.Debug, format+"\n", args...)
}

func (c *Client) composeArgs(rest ...string) []string {
	args := []string{"compose"}
	for _, p := range c.Profiles {
		args = append(args, "--profile", p)
	}
	args = append(args, "-p", c.Project, "-f", c.File)
	return append(args, rest...)
}

// Up starts the project detached. pull is always, missing, or never. With no
// services named it starts everything, and it is safe to call again to start the
// rest: compose leaves running containers alone.
func (c *Client) Up(ctx context.Context, pull string, services ...string) (Result, error) {
	args := c.composeArgs("up", "-d", "--quiet-pull")
	if pull != "" {
		args = append(args, "--pull", pull)
	}
	args = append(args, services...)
	res, err := c.run(ctx, nil, args...)
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("docker compose up failed: %s", firstLines(res.Combined(), 12))
	}
	return res, nil
}

// ExecOptions is one command run inside a running service.
type ExecOptions struct {
	Service string
	User    string
	Argv    []string
	Env     map[string]string
	Stdin   io.Reader
}

// Exec runs a command inside a service. The argv is passed through unchanged; a
// recipe's fields never reach a shell.
func (c *Client) Exec(ctx context.Context, o ExecOptions) (Result, error) {
	args := c.composeArgs("exec", "-T")
	if o.User != "" {
		args = append(args, "-u", o.User)
	}
	for k, v := range o.Env {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, o.Service)
	args = append(args, o.Argv...)
	return c.run(ctx, o.Stdin, args...)
}

// Logs returns the last n lines of one service's log.
func (c *Client) Logs(ctx context.Context, service string, tail int) (string, error) {
	res, err := c.run(ctx, nil, c.composeArgs(
		"logs", "--no-color", "--no-log-prefix", "--tail", fmt.Sprint(tail), service)...)
	if err != nil {
		return "", err
	}
	return res.Combined(), nil
}

// Services lists the services defined in the project's compose file.
func (c *Client) Services(ctx context.Context) ([]string, error) {
	res, err := c.run(ctx, nil, c.composeArgs("config", "--services")...)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("docker compose config failed: %s", firstLines(res.Combined(), 10))
	}
	var out []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// NetworkName returns the docker network compose created for this project. Checks run
// from a helper container attached to it, which is how drillback reaches an
// application it has deliberately not published a port for.
func (c *Client) NetworkName(ctx context.Context) (string, error) {
	res, err := c.run(ctx, nil, "network", "ls",
		"--filter", "label=com.docker.compose.project="+c.Project,
		"--format", "{{.Name}}")
	if err != nil {
		return "", err
	}
	names := strings.Fields(res.Stdout)
	if len(names) == 0 {
		return "", fmt.Errorf("no network for compose project %s", c.Project)
	}
	return names[0], nil
}

// RunOptions is a throwaway helper container on the run's network.
type RunOptions struct {
	Image   string
	Network string
	Argv    []string
	Timeout time.Duration
}

// RunHelper starts a container on the run's network, waits for it, and removes it.
func (c *Client) RunHelper(ctx context.Context, o RunOptions) (Result, error) {
	args := []string{"run", "--rm", "--network", o.Network,
		"--label", LabelRun + "=" + c.RunID,
		"--entrypoint", "",
		o.Image}
	args = append(args, o.Argv...)
	return c.run(ctx, nil, args...)
}

// CopyOut copies a path out of a running service into the local filesystem. It is
// how the harness exports a dir input: the daemon reads the files, so a tree the
// application wrote as its own user is readable without the caller being that user.
//
// docker cp semantics apply: when dest does not exist it is created and the source
// directory's *contents* are copied into it.
func (c *Client) CopyOut(ctx context.Context, service, src, dest string) error {
	res, err := c.run(ctx, nil, c.composeArgs("cp", service+":"+src, dest)...)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("docker compose cp %s:%s failed: %s", service, src, firstLines(res.Combined(), 8))
	}
	return nil
}

// Bind is one host path made visible inside a throwaway container.
type Bind struct {
	Host      string
	Container string
	ReadOnly  bool
}

func (b Bind) arg() string {
	s := b.Host + ":" + b.Container
	if b.ReadOnly {
		s += ":ro"
	}
	return s
}

// ContainerOptions is a throwaway container that is not part of the compose project:
// the harness uses one to drive restic against a repository inside the workspace.
//
// Env names are forwarded to the container by name only, so a value never appears in
// an argument list, a process listing, or the debug log.
type ContainerOptions struct {
	Image string
	Argv  []string
	Env   map[string]string
	Binds []Bind
	// User is the uid:gid the container runs as. It is set on Linux so that files
	// written into a bind mount belong to the caller rather than to root.
	User string
	// Network defaults to "none". A throwaway container that reads and writes only
	// the workspace has no reason to reach anything.
	Network string
}

// RunContainer runs a throwaway container to completion and removes it.
func (c *Client) RunContainer(ctx context.Context, o ContainerOptions) (Result, error) {
	network := o.Network
	if network == "" {
		network = "none"
	}
	args := []string{"run", "--rm", "--network", network, "--label", LabelRun + "=" + c.RunID}
	if o.User != "" {
		args = append(args, "--user", o.User)
	}
	names := make([]string, 0, len(o.Env))
	for k := range o.Env {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		args = append(args, "-e", k)
	}
	for _, b := range o.Binds {
		args = append(args, "-v", b.arg())
	}
	args = append(args, o.Image)
	args = append(args, o.Argv...)
	return c.runEnv(ctx, o.Env, nil, args...)
}

// Down removes the containers, the volumes, and the network. It is idempotent and is
// safe to call when nothing was ever created.
func (c *Client) Down(ctx context.Context) error {
	res, err := c.run(ctx, nil, c.composeArgs(
		"down", "-v", "--remove-orphans", "--timeout", "20")...)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("docker compose down failed: %s", firstLines(res.Combined(), 10))
	}
	return nil
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + "\n..."
}
