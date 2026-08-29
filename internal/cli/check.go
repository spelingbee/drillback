package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/spelingbee/restored/internal/nudge"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/recipe/safety"
	"github.com/spelingbee/restored/internal/report"
	"github.com/spelingbee/restored/internal/runner"
)

type checkFlags struct {
	recipeRef string
	source    string
	from      string
	snapshot  string
	tags      []string
	host      string
	inputs    []string
	sets      []string

	timeout        time.Duration
	restoreTimeout time.Duration
	readyTimeout   time.Duration
	checkTimeout   time.Duration

	pull       string
	workspace  string
	keep       bool
	keepOnFail bool
	reportFile string
	hintsFile  string
	noNudge    bool
}

func newCheck(g *globals) *cobra.Command {
	f := &checkFlags{}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Restore a backup and verify that the application boots",
		Long: "Restore a backup into a throwaway environment and verify that the application boots.\n\n" +
			"Every run gets its own workspace directory, its own compose project named\n" +
			"restored-<runid>, and its own internal network. No ports are published; HTTP checks\n" +
			"run from a helper container attached to the run's network. The workspace and the\n" +
			"compose project are always destroyed on exit — including on Ctrl-C and on panic —\n" +
			"unless --keep or --keep-on-fail says otherwise.",
		Example: "  # a local recipe directory, against a tree that is already restored on disk\n" +
			"  restored check --recipe ./recipes/uptime-kuma --source dir --from /mnt/export/uk\n\n" +
			"  # restic repository from the environment, bundled recipe, latest snapshot\n" +
			"  export RESTIC_REPOSITORY=/mnt/backups/restic\n" +
			"  export RESTIC_PASSWORD_FILE=/etc/restic/pass\n" +
			"  restored check --recipe gitea",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runCheck(cmd, g, f) },
	}
	fl := cmd.Flags()
	fl.StringVar(&f.recipeRef, "recipe", "", `Recipe to run: a bundled name (e.g. "gitea"), a path to a directory containing recipe.yaml, or a path to a recipe.yaml file`)
	fl.StringVar(&f.source, "source", "restic", "Backup source: restic|dir")
	fl.StringVar(&f.from, "from", "", "Source location: the restic repository, or the already-restored tree")
	fl.StringVar(&f.snapshot, "snapshot", "latest", `restic snapshot id, or "latest"`)
	fl.StringSliceVar(&f.tags, "tag", nil, "Only consider snapshots carrying this tag (repeatable)")
	fl.StringVar(&f.host, "host", "", "Only consider snapshots written by this host")
	fl.StringArrayVar(&f.inputs, "input", nil, "Override a recipe input's path inside the backup (repeatable), as name=path")
	fl.StringArrayVar(&f.sets, "set", nil, "Override a recipe variable (repeatable), as key=value")
	fl.DurationVar(&f.timeout, "timeout", 15*time.Minute, "Wall-clock budget for the whole run")
	fl.DurationVar(&f.restoreTimeout, "restore-timeout", 10*time.Minute, "Budget for the restore stage")
	fl.DurationVar(&f.readyTimeout, "ready-timeout", 5*time.Minute, "Budget for all ready probes together")
	fl.DurationVar(&f.checkTimeout, "check-timeout", 60*time.Second, "Per-check timeout")
	fl.StringVar(&f.pull, "pull", "missing", "Image pull policy: always|missing|never")
	fl.StringVar(&f.workspace, "workspace", "", "Parent directory for the run workspace (default: the OS temp directory)")
	fl.BoolVar(&f.keep, "keep", false, "Do not tear down; print the workspace path and the compose project name")
	fl.BoolVar(&f.keepOnFail, "keep-on-fail", false, "Tear down on PASS, keep everything on failure")
	fl.StringVar(&f.reportFile, "report", "", "Also write the JSON report to this file")
	fl.StringVar(&f.hintsFile, "hints", "", "Load additional hint rules, matched before the built-in ones")
	fl.BoolVar(&f.noNudge, "no-nudge", false, `Never print the "contribute this recipe" invitation`)
	return cmd
}

func runCheck(cmd *cobra.Command, g *globals, f *checkFlags) error {
	if f.recipeRef == "" {
		return fail(ExitError, "--recipe is required")
	}
	inputs, err := keyValues(f.inputs, "input")
	if err != nil {
		return fail(ExitError, "%v", err)
	}
	sets, err := keyValues(f.sets, "set")
	if err != nil {
		return fail(ExitError, "%v", err)
	}
	switch f.pull {
	case "always", "missing", "never":
	default:
		return fail(ExitError, "--pull %q: expected always, missing or never", f.pull)
	}

	rec, err := recipe.LoadAny(f.recipeRef)
	if err != nil {
		return fail(ExitError, "%v", err)
	}

	var debug *os.File
	if g.logLevel == "debug" || g.logLevel == "trace" {
		debug = os.Stderr
	}

	opts := runner.Options{
		Recipe:          rec,
		SourceKind:      f.source,
		From:            f.from,
		Snapshot:        f.snapshot,
		Tags:            f.tags,
		Host:            f.host,
		InputPaths:      inputs,
		SetVars:         sets,
		Timeout:         f.timeout,
		RestoreTimeout:  f.restoreTimeout,
		ReadyTimeout:    f.readyTimeout,
		CheckTimeout:    f.checkTimeout,
		Pull:            f.pull,
		WorkspaceParent: f.workspace,
		Keep:            f.keep,
		KeepOnFail:      f.keepOnFail,
		HintsFile:       f.hintsFile,
		Version:         Version,
		Commit:          Commit,
	}
	if debug != nil {
		opts.Debug = debug
	}

	rep, kept, runErr := runner.Run(cmd.Context(), opts)

	human := cmd.OutOrStdout()
	if g.json {
		human = cmd.ErrOrStderr()
	}
	if rep != nil {
		renderOpts := report.Options{
			Color: colourEnabled(g, human),
			ASCII: os.Getenv("RESTORED_ASCII") != "",
		}
		if runErr == nil {
			if err := rep.WriteTTY(human, renderOpts); err != nil {
				return fail(ExitError, "%v", err)
			}
		}
		if g.json {
			if err := rep.WriteJSON(cmd.OutOrStdout()); err != nil {
				return fail(ExitError, "%v", err)
			}
		}
		if f.reportFile != "" {
			file, err := os.Create(f.reportFile)
			if err != nil {
				return fail(ExitError, "writing --report: %v", err)
			}
			writeErr := rep.WriteJSON(file)
			closeErr := file.Close()
			if writeErr != nil {
				return fail(ExitError, "%v", writeErr)
			}
			if closeErr != nil {
				return fail(ExitError, "writing --report: %v", closeErr)
			}
		}
	}
	if kept != nil {
		fmt.Fprintf(human, "\nKept for inspection:\n  workspace:        %s\n  compose project:  %s\n\n"+
			"Clean up with:\n  docker compose -p %s down -v --remove-orphans\n  rm -rf %s\n",
			kept.Workspace, kept.Project, kept.Project, kept.Workspace)
	}
	if runErr != nil {
		return fail(ExitError, "%v", runErr)
	}
	if rep.Verdict != report.VerdictPass {
		return &exitError{code: ExitUnusable, err: errSilent{}}
	}

	maybeNudge(cmd, g, f, rec, inputs, sets, human)
	return nil
}

// errSilent is the error for a verdict that has already been printed in full.
type errSilent struct{}

func (errSilent) Error() string { return "" }

// maybeNudge prints the contribution invitation, but only when every condition in
// SPEC.md section 8.1 holds. Nudging someone toward a PR that CI will immediately
// reject wastes their goodwill, which is the scarcest resource this project has.
func maybeNudge(cmd *cobra.Command, g *globals, f *checkFlags, rec *recipe.Recipe,
	inputs, sets map[string]string, w interface{ Write([]byte) (int, error) }) {
	if g.noNudge || f.noNudge || g.json || rec.Bundled {
		return
	}
	if !isTerminal(cmd.OutOrStdout()) && !isTerminal(cmd.ErrOrStderr()) {
		return
	}
	composeRaw, err := rec.ReadFile("compose.yaml")
	if err != nil {
		return
	}
	res, err := recipe.Resolve(rec, recipe.Options{InputsDir: "/workspace/inputs", RunID: "nudge"})
	if err != nil {
		return
	}
	if err := safety.Validate(composeRaw, res); err != nil {
		return
	}
	if len(safety.Warnings(rec, composeRaw)) > 0 {
		return
	}
	folded, err := nudge.FoldOverrides(rec.Raw, sets, inputs)
	if err != nil {
		return
	}
	_, _ = fmt.Fprint(w, nudge.Build(nudge.Input{
		Name:  rec.Metadata.Name,
		YAML:  folded,
		Path:  rec.File,
		Title: rec.Metadata.Title,
	}))
}

func colourEnabled(g *globals, w any) bool {
	if g.noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return isTerminal(w)
}

func isTerminal(w any) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
