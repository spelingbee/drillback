package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spelingbee/drillback/internal/config"
)

// The load-bearing claim of ADR-068: a flag beats the config only when the user
// actually typed it, and a flag left at its default is not an opinion. This walks
// jobFromConfig field by field, with typed and untyped flags side by side.
func TestJobFromConfigPrecedence(t *testing.T) {
	cfg := writePrecedenceConfig(t)
	load := func(t *testing.T) *config.Job {
		t.Helper()
		cj, err := cfg.Job("gitea")
		if err != nil {
			t.Fatal(err)
		}
		return cj
	}

	t.Run("typed flag beats the target, untyped default loses to the config", func(t *testing.T) {
		cmd, f := newCheckCommand(&globals{})
		if err := cmd.Flags().Set("check-timeout", "45s"); err != nil {
			t.Fatal(err)
		}
		j, err := jobFromConfig(cmd, f, load(t), map[string]string{"db": "/cli/db"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if j.opts.CheckTimeout != 45*time.Second {
			t.Errorf("check timeout = %v, want the typed flag's 45s over the target's 90s", j.opts.CheckTimeout)
		}
		if j.opts.Timeout != 15*time.Minute {
			t.Errorf("timeout = %v, want the config's 15m over the untyped flag default 30m", j.opts.Timeout)
		}
		if j.opts.RestoreTimeout != 10*time.Minute {
			t.Errorf("restore timeout = %v, want the flag default where the config is silent", j.opts.RestoreTimeout)
		}
		if j.opts.SourceKind != "restic" || j.opts.From != "/srv/restic" {
			t.Errorf("source = %s %s, want the config's", j.opts.SourceKind, j.opts.From)
		}
		if j.opts.Host != "hypervisor" {
			t.Errorf("host = %q, want the source's default filter", j.opts.Host)
		}
		if len(j.opts.Tags) != 1 || j.opts.Tags[0] != "gitea" {
			t.Errorf("tags = %v, want the target's", j.opts.Tags)
		}
		if j.opts.InputPaths["data"] != "/srv/gitea/data" || j.opts.InputPaths["db"] != "/cli/db" {
			t.Errorf("inputs = %v, want config's data and the CLI's db", j.opts.InputPaths)
		}
		if want := "RESTIC_PASSWORD_FILE=" + filepath.Join(filepath.Dir(cfg.Path), "nas.pass"); !containsString(j.opts.SourceEnv, want) {
			t.Errorf("source env %v missing %q", j.opts.SourceEnv, want)
		}
	})

	t.Run("typed --tag replaces the target's tags", func(t *testing.T) {
		cmd, f := newCheckCommand(&globals{})
		if err := cmd.Flags().Set("tag", "other"); err != nil {
			t.Fatal(err)
		}
		j, err := jobFromConfig(cmd, f, load(t), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(j.opts.Tags) != 1 || j.opts.Tags[0] != "other" {
			t.Errorf("tags = %v, want the typed flag's", j.opts.Tags)
		}
	})

	t.Run("--source alone refuses, naming the cure", func(t *testing.T) {
		cmd, f := newCheckCommand(&globals{})
		if err := cmd.Flags().Set("source", "dir"); err != nil {
			t.Fatal(err)
		}
		_, err := jobFromConfig(cmd, f, load(t), nil, nil)
		if err == nil || !strings.Contains(err.Error(), "--from") {
			t.Errorf("a typed --source without --from must refuse and name --from: %v", err)
		}
	})

	t.Run("--source with --from replaces the whole triple and drops the config's env", func(t *testing.T) {
		cmd, f := newCheckCommand(&globals{})
		for flag, v := range map[string]string{"source": "dir", "from": "/mnt/export"} {
			if err := cmd.Flags().Set(flag, v); err != nil {
				t.Fatal(err)
			}
		}
		j, err := jobFromConfig(cmd, f, load(t), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if j.opts.SourceKind != "dir" || j.opts.From != "/mnt/export" {
			t.Errorf("source = %s %s", j.opts.SourceKind, j.opts.From)
		}
		if len(j.opts.SourceEnv) != 0 {
			t.Errorf("a replaced source must not carry the config source's env: %v", j.opts.SourceEnv)
		}
	})

	t.Run("--from alone repoints the config's source and keeps its env", func(t *testing.T) {
		cmd, f := newCheckCommand(&globals{})
		if err := cmd.Flags().Set("from", "/mnt/other-repo"); err != nil {
			t.Fatal(err)
		}
		j, err := jobFromConfig(cmd, f, load(t), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if j.opts.SourceKind != "restic" || j.opts.From != "/mnt/other-repo" {
			t.Errorf("source = %s %s", j.opts.SourceKind, j.opts.From)
		}
		if len(j.opts.SourceEnv) == 0 {
			t.Error("repointing the same source must keep its env")
		}
	})
}

// An interrupted sweep proves nothing about the targets it never reached, and before
// the fix it could exit 0 claiming a clean run. The cancelled context breaks the loop
// before the first target, so this runs without docker.
func TestRunAllInterruptedIsUnprovenNotClean(t *testing.T) {
	cfg := writePrecedenceConfig(t)
	cj, err := cfg.Job("gitea")
	if err != nil {
		t.Fatal(err)
	}
	g := &globals{}
	cmd, f := newCheckCommand(g)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	j, err := jobFromConfig(cmd, f, cj, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = runAll(cmd, g, f, []*job{j, j})
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != ExitError {
		t.Fatalf("interrupted --all returned %v; want exit %d - what never ran is unproven", err, ExitError)
	}
	if !strings.Contains(out.String(), "2 of 2 targets never ran") {
		t.Errorf("the summary must say what never ran:\n%s", out.String())
	}
}

func writePrecedenceConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "drillback.yaml")
	body := `
version: 1
defaults:
  source: nas
  timeout: 15m
sources:
  nas:
    kind: restic
    repository: /srv/restic
    password_file: nas.pass
    host: hypervisor
targets:
  gitea:
    recipe: gitea
    tags: [gitea]
    inputs:
      data: /srv/gitea/data
    check_timeout: 90s
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
