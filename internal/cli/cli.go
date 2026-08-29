// Package cli is the only package that knows about exit codes, and the only one that
// writes to stdout and stderr directly. See SPEC.md section 13.1.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// Build information, set with -ldflags at release time.
var (
	Version = "0.1.0-dev"
	Commit  = ""
	Date    = ""
)

// Exit codes. The 1/2 split is the contract a cron job depends on: 1 says the backup
// is broken, 2 says the drill is broken. See SPEC.md section 5.3.
const (
	ExitPass        = 0
	ExitUnusable    = 1
	ExitError       = 2
	ExitInterrupted = 130
)

// globals are the flags every command shares.
type globals struct {
	json     bool
	noColor  bool
	noNudge  bool
	logLevel string
}

// exitError carries an exit code out of a command.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

func fail(code int, format string, args ...any) error {
	return &exitError{code: code, err: fmt.Errorf(format, args...)}
}

// Main runs the CLI and returns the process exit code.
func Main() int {
	ctx, stop := interruptContext()
	defer stop()

	root := newRoot()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	err := root.ExecuteContext(ctx)
	if err == nil {
		return ExitPass
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.err != nil && strings.TrimSpace(ee.err.Error()) != "" {
			fmt.Fprintf(os.Stderr, "restored: %v\n", ee.err)
		}
		return ee.code
	}
	fmt.Fprintf(os.Stderr, "restored: %v\n", err)
	return ExitError
}

// interruptContext cancels on the first SIGINT or SIGTERM so teardown can run, and
// leaves immediately on the second, after saying what was left behind.
func interruptContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		fmt.Fprintln(os.Stderr, "\nrestored: interrupted, tearing down (press Ctrl-C again to leave everything in place)")
		cancel()
		<-ch
		fmt.Fprintln(os.Stderr, "restored: leaving the workspace and the compose project in place")
		os.Exit(ExitInterrupted)
	}()
	return ctx, func() {
		signal.Stop(ch)
		cancel()
	}
}

func newRoot() *cobra.Command {
	g := &globals{}
	root := &cobra.Command{
		Use:   "restored",
		Short: "restored — your backup is a lie until it boots.",
		Long: "restored — your backup is a lie until it boots.\n\n" +
			"Restores the latest snapshot of a backup into a throwaway, isolated environment,\n" +
			"starts the application with docker compose, and verifies that it actually works.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	root.PersistentFlags().BoolVar(&g.json, "json", false,
		"Emit the machine-readable report on stdout; human output goes to stderr")
	root.PersistentFlags().StringVar(&g.logLevel, "log-level", "info", "trace|debug|info|warn|error")
	root.PersistentFlags().BoolVar(&g.noColor, "no-color", false, "Disable ANSI colour (NO_COLOR is also honoured)")
	root.PersistentFlags().BoolVar(&g.noNudge, "no-nudge", false, `Never print the "contribute this recipe" invitation`)

	root.AddCommand(newCheck(g), newRecipe(g), newVersion(g))
	root.SetHelpTemplate(root.HelpTemplate() + `
Exit codes:
  0   all checks passed
  1   restore unusable — one or more checks failed, or the app never became ready
  2   tool or runtime error — docker missing, restic failed, recipe invalid, timeout
      before any check could run

Docs: https://github.com/spelingbee/restored
`)
	return root
}

// keyValues parses repeatable name=value flags.
func keyValues(pairs []string, flag string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("--%s %q: expected name=value", flag, p)
		}
		out[k] = v
	}
	return out, nil
}
