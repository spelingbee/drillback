// Package loader makes a restored dump live: a PostgreSQL dump is loaded into its
// service, and a SQLite database is verified in place.
package loader

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spelingbee/restored/internal/compose"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/sqlite"
)

// Dump formats, as detected from the file rather than from its extension.
const (
	FormatPlain  = "plain"
	FormatCustom = "custom"
	FormatEmpty  = "empty"
)

// Detail is what the report says about loading one input.
type Detail struct {
	Loader      string `json:"loader,omitempty"`
	Format      string `json:"format,omitempty"`
	StderrLines int    `json:"stderr_lines"`
	Error       string `json:"error,omitempty"`
}

// DetectFormat reads the first five bytes. `pg_dump -Fc`, `-Fd` and `-Ft` all start
// with PGDMP; anything else is treated as plain SQL. The detected format goes into the
// report so a surprise is visible rather than mysterious.
func DetectFormat(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening the dump: %w", err)
	}
	defer f.Close()
	head := make([]byte, 5)
	n, err := f.Read(head)
	if n == 0 {
		return FormatEmpty, nil
	}
	if err != nil && n < 5 {
		return FormatPlain, nil
	}
	if string(head[:n]) == "PGDMP" {
		return FormatCustom, nil
	}
	return FormatPlain, nil
}

// LoadPostgres streams a dump into its service. A dump that will not load is a verdict
// about the backup, not a tool error: the caller maps it to exit 1.
func LoadPostgres(ctx context.Context, cli *compose.Client, in *recipe.ResolvedInput, timeout time.Duration) (Detail, error) {
	if in.Load == nil {
		return Detail{}, fmt.Errorf("input %q is a postgres-dump with no load block", in.Name)
	}
	format, err := DetectFormat(in.LocalPath)
	if err != nil {
		return Detail{}, err
	}
	d := Detail{Format: format}
	if format == FormatEmpty {
		d.Error = "the dump file is empty (0 bytes)"
		return d, fmt.Errorf("input %q: %s", in.Name, d.Error)
	}

	var argv []string
	switch format {
	case FormatCustom:
		d.Loader = "pg_restore"
		argv = []string{"pg_restore", "--clean", "--if-exists", "--no-owner", "--no-acl",
			"-U", in.Load.User, "-d", in.Load.Database}
	default:
		d.Loader = "psql"
		argv = []string{"psql", "-v", "ON_ERROR_STOP=1", "--quiet",
			"-U", in.Load.User, "-d", in.Load.Database}
	}

	f, err := os.Open(in.LocalPath)
	if err != nil {
		return d, fmt.Errorf("opening the dump: %w", err)
	}
	defer f.Close()

	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := cli.Exec(runCtx, compose.ExecOptions{
		Service: in.Load.Service,
		Argv:    argv,
		Stdin:   f,
	})
	if err != nil {
		return d, err
	}
	d.StderrLines = countLines(res.Stderr)
	if res.ExitCode != 0 {
		d.Error = tail(res.Combined(), 20)
		return d, fmt.Errorf("loading input %q with %s: exit %d: %s",
			in.Name, d.Loader, res.ExitCode, d.Error)
	}
	return d, nil
}

// IntegrityCheck runs PRAGMA integrity_check against a restored SQLite file. A
// malformed database is a restore failure, not a tool error.
func IntegrityCheck(ctx context.Context, in *recipe.ResolvedInput) (Detail, error) {
	d := Detail{Loader: "sqlite integrity_check"}
	rows, err := sqlite.Query(ctx, in.LocalPath, "PRAGMA integrity_check;")
	if err != nil {
		d.Error = err.Error()
		return d, fmt.Errorf("input %q: %w", in.Name, err)
	}
	got := ""
	if len(rows) > 0 && len(rows[0]) > 0 {
		got = rows[0][0]
	}
	if !strings.EqualFold(strings.TrimSpace(got), "ok") {
		d.Error = got
		return d, fmt.Errorf("input %q: PRAGMA integrity_check returned %q", in.Name, got)
	}
	return d, nil
}

func countLines(s string) int {
	s = strings.TrimRight(s, "\r\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
