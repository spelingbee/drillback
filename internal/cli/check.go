package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/spelingbee/restored/internal/config"
	"github.com/spelingbee/restored/internal/nudge"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/recipe/safety"
	"github.com/spelingbee/restored/internal/report"
	"github.com/spelingbee/restored/internal/runner"
)

type checkFlags struct {
	recipeRef string
	target    string
	all       bool

	source   string
	from     string
	snapshot string
	tags     []string
	host     string
	inputs   []string
	sets     []string

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
	cmd, _ := newCheckCommand(g)
	return cmd
}

// newCheckCommand also returns the flag struct, which is how the precedence tests
// reach jobFromConfig with the exact flag set a user's invocation binds.
func newCheckCommand(g *globals) (*cobra.Command, *checkFlags) {
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
			"  restored check --recipe gitea\n\n" +
			"  # one named target from restored.yaml\n" +
			"  restored check --target gitea\n\n" +
			"  # every target in restored.yaml, for cron\n" +
			"  restored check --all --json --report /var/log/restored/run.json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runCheck(cmd, g, f) },
	}
	fl := cmd.Flags()
	fl.StringVar(&f.recipeRef, "recipe", "", `Recipe to run: a bundled name (e.g. "gitea"), a path to a directory containing recipe.yaml, or a path to a recipe.yaml file. Required unless --target or --all`)
	fl.StringVar(&f.target, "target", "", "Run one named target from restored.yaml")
	fl.BoolVar(&f.all, "all", false, "Run every target in restored.yaml, sequentially")
	fl.StringVar(&f.source, "source", "restic", "Backup source: restic|dir")
	fl.StringVar(&f.from, "from", "", "Source location: the restic repository, or the already-restored tree")
	fl.StringVar(&f.snapshot, "snapshot", "latest", `restic snapshot id, or "latest"`)
	fl.StringSliceVar(&f.tags, "tag", nil, "Only consider snapshots carrying this tag (repeatable)")
	fl.StringVar(&f.host, "host", "", "Only consider snapshots written by this host")
	fl.StringArrayVar(&f.inputs, "input", nil, "Override a recipe input's path inside the backup (repeatable), as name=path")
	fl.StringArrayVar(&f.sets, "set", nil, "Override a recipe variable (repeatable), as key=value")
	// 30 minutes, not 15: --restore-timeout and --ready-timeout below already sum to
	// 15, so the old default guaranteed that a large restore left nothing for the
	// stages after it. A run that shortens this has its stage budgets clamped to fit
	// inside it; see runner.applyDefaults and DECISIONS.md ADR-058.
	fl.DurationVar(&f.timeout, "timeout", 30*time.Minute, "Wall-clock budget for the whole run, including every stage below")
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
	AddExitCodes(cmd, CheckExitCodes)
	return cmd, f
}

// job carries one resolved run: where its settings came from is already decided.
type job struct {
	target string // "" when the run came from --recipe
	rec    *recipe.Recipe
	opts   runner.Options
	inputs map[string]string
	sets   map[string]string
}

func runCheck(cmd *cobra.Command, g *globals, f *checkFlags) error {
	modes := 0
	for _, on := range []bool{f.recipeRef != "", f.target != "", f.all} {
		if on {
			modes++
		}
	}
	switch {
	case modes > 1:
		return fail(ExitError, "--recipe, --target and --all are mutually exclusive")
	case modes == 0:
		return fail(ExitError, "--recipe is required (or --target <name> / --all, which read restored.yaml)")
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

	jobs, err := resolveJobs(cmd, g, f, inputs, sets)
	if err != nil {
		return err
	}

	if f.all {
		return runAll(cmd, g, f, jobs)
	}
	return runSingle(cmd, g, f, jobs[0])
}

// resolveJobs turns the invocation into runnable jobs, all of them before any run
// starts: a typo in the fifth target of restored.yaml should refuse now, not at four
// in the morning after the first four targets already ran.
func resolveJobs(cmd *cobra.Command, g *globals, f *checkFlags, inputs, sets map[string]string) ([]*job, error) {
	if f.recipeRef != "" {
		rec, err := recipe.LoadAny(f.recipeRef)
		if err != nil {
			return nil, fail(ExitError, "%v", err)
		}
		return []*job{{
			rec:    rec,
			inputs: inputs,
			sets:   sets,
			opts: runner.Options{
				Recipe:     rec,
				SourceKind: f.source, From: f.from, Snapshot: f.snapshot, Tags: f.tags, Host: f.host,
				InputPaths: inputs, SetVars: sets,
				Timeout: f.timeout, RestoreTimeout: f.restoreTimeout,
				ReadyTimeout: f.readyTimeout, CheckTimeout: f.checkTimeout,
				Pull: f.pull, WorkspaceParent: f.workspace,
				Keep: f.keep, KeepOnFail: f.keepOnFail, HintsFile: f.hintsFile,
				Version: Version, Commit: Commit,
			},
		}}, nil
	}

	path, err := config.Discover(g.configPath)
	if err != nil {
		return nil, fail(ExitError, "%v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fail(ExitError, "%v", err)
	}

	names := []string{f.target}
	if f.all {
		names = cfg.EnabledTargets()
		if len(names) == 0 {
			return nil, fail(ExitError, "%s: --all found no enabled targets", path)
		}
	}

	jobs := make([]*job, 0, len(names))
	for _, name := range names {
		cj, err := cfg.Job(name)
		if err != nil {
			return nil, fail(ExitError, "%v", err)
		}
		j, err := jobFromConfig(cmd, f, cj, inputs, sets)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// jobFromConfig finishes the precedence chain of SPEC.md 2.9: the config resolved
// recipe defaults, `defaults:` and the target block; a flag beats all of that, but
// only a flag the user actually typed - a flag left at its default is not an opinion.
func jobFromConfig(cmd *cobra.Command, f *checkFlags, cj *config.Job, cliInputs, cliSets map[string]string) (*job, error) {
	changed := cmd.Flags().Changed

	rec, err := recipe.LoadAny(cj.RecipeRef)
	if err != nil {
		return nil, fail(ExitError, "target %q: %v", cj.Target, err)
	}

	pick := func(flag, flagVal, cfgVal string) string {
		if changed(flag) || cfgVal == "" {
			return flagVal
		}
		return cfgVal
	}
	pickD := func(flag string, flagVal time.Duration, cfgVal time.Duration) time.Duration {
		if changed(flag) || cfgVal == 0 {
			return flagVal
		}
		return cfgVal
	}

	inputs := merged(cj.Inputs, cliInputs)
	sets := merged(cj.Set, cliSets)
	tags := cj.Tags
	if changed("tag") {
		tags = f.tags
	}

	// The source is one compound setting - kind, location, credentials - and
	// replacing only its kind produces a job that fails downstream with an error
	// that never says why ("no such directory sftp:..."). A typed --source replaces
	// the whole triple, so it needs a typed --from and drops the config source's
	// env; a typed --from alone repoints the config's source and keeps the rest.
	sourceKind, from, sourceEnv := cj.SourceKind, cj.From, cj.Env
	if changed("source") {
		if !changed("from") {
			return nil, fail(ExitError,
				"target %q: --source replaces the config's source, so it needs --from too", cj.Target)
		}
		sourceKind, from, sourceEnv = f.source, f.from, nil
	} else if changed("from") {
		from = f.from
	}

	opts := runner.Options{
		Recipe:     rec,
		SourceKind: sourceKind,
		From:       from,
		Snapshot:   f.snapshot,
		Tags:       tags,
		Host:       pick("host", f.host, cj.Host),
		SourceEnv:  sourceEnv,
		InputPaths: inputs,
		SetVars:    sets,

		Timeout:        pickD("timeout", f.timeout, cj.Timeout),
		RestoreTimeout: pickD("restore-timeout", f.restoreTimeout, cj.RestoreTimeout),
		ReadyTimeout:   pickD("ready-timeout", f.readyTimeout, cj.ReadyTimeout),
		CheckTimeout:   pickD("check-timeout", f.checkTimeout, cj.CheckTimeout),

		Pull:            pick("pull", f.pull, cj.Pull),
		WorkspaceParent: pick("workspace", f.workspace, cj.Workspace),
		Keep:            f.keep,
		KeepOnFail:      f.keepOnFail,
		HintsFile:       f.hintsFile,
		Version:         Version,
		Commit:          Commit,
	}
	return &job{target: cj.Target, rec: rec, opts: opts, inputs: inputs, sets: sets}, nil
}

// merged overlays CLI name=value pairs on the config's: the flag wins per key.
func merged(cfg, cli map[string]string) map[string]string {
	if len(cfg) == 0 {
		return cli
	}
	out := make(map[string]string, len(cfg)+len(cli))
	for k, v := range cfg {
		out[k] = v
	}
	for k, v := range cli {
		out[k] = v
	}
	return out
}

// runSingle is `--recipe` and `--target`: one run, the single-run JSON document, and
// the exit code straight from the verdict.
func runSingle(cmd *cobra.Command, g *globals, f *checkFlags, j *job) error {
	j.opts.Debug = debugWriter(g)

	rep, kept, runErr := runner.Run(cmd.Context(), j.opts)

	human := cmd.OutOrStdout()
	if g.json {
		human = cmd.ErrOrStderr()
	}
	renderOpts := report.Options{
		Color: colourEnabled(g, human),
		ASCII: os.Getenv("RESTORED_ASCII") != "",
	}
	if rep != nil {
		// Printed on a tool error too, not only on a verdict. A run that got far
		// enough to have an id got far enough to be worth showing: the stages it
		// completed, the workspace it used, and - since ADR-058 - the hint that says
		// what to do next. Before this, a tool error printed one line and no report,
		// while --report still wrote the JSON, so a machine saw more than the human.
		if rep.Run.ID != "" {
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
			if err := writeReportFile(f.reportFile, rep.WriteJSON); err != nil {
				return err
			}
		}
	}
	printKept(human, kept)
	if runErr != nil {
		// The report already printed the error, in a block with the stages that led
		// to it and the hint that says what to do next. Repeating it underneath is
		// noise. When there was no report - a failure before the run had an id -
		// this line is the only thing the user gets, so it still has to be here.
		if rep != nil && rep.Run.ID != "" {
			return &exitError{code: ExitError, err: errSilent{}}
		}
		return fail(ExitError, "%v", runErr)
	}
	if rep.Verdict != report.VerdictPass {
		return &exitError{code: ExitUnusable, err: errSilent{}}
	}

	maybeNudge(cmd, g, f, j.rec, j.inputs, j.sets, human)
	return nil
}

// runAll is `--all`: every enabled target in file order, each rendered as it
// finishes, then the summary block and the multi-target document of SPEC.md 5.2.
// The exit code is the worst outcome across targets. No nudge: an invitation is
// for a person at a terminal trying one recipe, not for a cron sweep.
func runAll(cmd *cobra.Command, g *globals, f *checkFlags, jobs []*job) error {
	human := cmd.OutOrStdout()
	if g.json {
		human = cmd.ErrOrStderr()
	}
	renderOpts := report.Options{
		Color: colourEnabled(g, human),
		ASCII: os.Getenv("RESTORED_ASCII") != "",
	}

	multi := report.NewMulti()
	for i, j := range jobs {
		if cmd.Context().Err() != nil {
			// Interrupted: the targets already run are the answer there is.
			break
		}
		if i > 0 {
			fmt.Fprintln(human)
		}
		fmt.Fprintf(human, "target %s\n", j.target)

		j.opts.Debug = debugWriter(g)
		rep, kept, runErr := runner.Run(cmd.Context(), j.opts)
		if rep == nil {
			// runner.Run never does this today; the guard keeps a future change
			// from turning one broken target into a nil dereference at 04:00.
			rep = &report.Report{Verdict: report.VerdictError, ExitCode: ExitError}
			if runErr != nil {
				rep.Error = runErr.Error()
			}
		}
		if rep.Run.ID != "" {
			if err := rep.WriteTTY(human, renderOpts); err != nil {
				return fail(ExitError, "%v", err)
			}
		} else if runErr != nil {
			fmt.Fprintf(human, "restored: %v\n", runErr)
		}
		printKept(human, kept)
		multi.Add(j.target, rep)
	}
	// An interrupt between targets leaves the rest of the sweep unrun. Say so, in
	// both documents, and exit 2: a target that never ran is unproven, and without
	// this a SIGTERM after a passing teardown reported a clean sweep and the cron
	// alert never fired for the backups nobody checked.
	multi.SkipRemaining(len(jobs) - multi.Summary.TargetsTotal)

	if err := multi.WriteTTY(human, renderOpts); err != nil {
		return fail(ExitError, "%v", err)
	}
	if g.json {
		if err := multi.WriteJSON(cmd.OutOrStdout()); err != nil {
			return fail(ExitError, "%v", err)
		}
	}
	if f.reportFile != "" {
		if err := writeReportFile(f.reportFile, multi.WriteJSON); err != nil {
			return err
		}
	}

	if multi.ExitCode != 0 {
		// Every error and every verdict has already been printed, per target.
		return &exitError{code: multi.ExitCode, err: errSilent{}}
	}
	return nil
}

func printKept(human interface{ Write([]byte) (int, error) }, kept *runner.Kept) {
	if kept == nil {
		return
	}
	fmt.Fprintf(human, "\nKept for inspection:\n  workspace:        %s\n  compose project:  %s\n\n"+
		"Clean up with:\n  docker compose -p %s down -v --remove-orphans\n  rm -rf %s\n",
		kept.Workspace, kept.Project, kept.Project, kept.Workspace)
}

func writeReportFile(path string, write func(w io.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return fail(ExitError, "writing --report: %v", err)
	}
	writeErr := write(file)
	closeErr := file.Close()
	if writeErr != nil {
		return fail(ExitError, "%v", writeErr)
	}
	if closeErr != nil {
		return fail(ExitError, "writing --report: %v", closeErr)
	}
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
	// `rec.Bundled` is false for a recipe loaded from a path, so `--recipe
	// ./recipes/gitea` - which is what scripts/demo.sh does, and what anyone who
	// copied a bundled recipe to edit one line does - was invited to contribute a
	// recipe that already ships. The registry is the question, not how it was loaded.
	for _, name := range recipe.BundledNames() {
		if name == rec.Metadata.Name {
			return
		}
	}
	// `nudge: false` in restored.yaml is the config equivalent of --no-nudge.
	if config.NudgeSilenced(g.configPath) {
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
