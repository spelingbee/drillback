package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spelingbee/drillback/internal/check"
	"github.com/spelingbee/drillback/internal/compose"
	"github.com/spelingbee/drillback/internal/probe"
	"github.com/spelingbee/drillback/internal/recipe"
	"github.com/spelingbee/drillback/internal/recipe/safety"
	"github.com/spelingbee/drillback/internal/report"
	"github.com/spelingbee/drillback/internal/runner"
	"github.com/spelingbee/drillback/internal/workspace"
)

// exportMount is where ${DRILLBACK_EXPORT} appears inside every service during stage
// B. A step writes its artifact there; the harness collects it from the host side.
const exportMount = "/export"

// stageB is the round trip: start a fresh stack with empty inputs, let the
// application create its own world, seed it, export what a backup would have taken,
// put that into a throwaway restic repository, tear everything down, and then run the
// ordinary `drillback check` against the repository.
//
// Step 8 is the point of the design: the stage ends by running exactly the code path
// a user runs, so the harness cannot pass while the real path is broken.
func (o Options) stageB(ctx context.Context, b budget) (st Stage, kept *Kept, err error) {
	st = Stage{Name: "B", Title: "round trip: seed, export, back up, restore, check"}
	start := time.Now()
	defer func() { st.DurationMS = time.Since(start).Milliseconds() }()

	ws, err := workspace.New(o.WorkspaceParent)
	if err != nil {
		return failStage(st, err)
	}

	debug := o.Debug
	logFile, logErr := os.Create(filepath.Join(ws.LogsDir(), "harness.log"))
	if logErr == nil {
		if debug == nil {
			debug = logFile
		} else {
			debug = io.MultiWriter(debug, logFile)
		}
	}

	cli := &compose.Client{
		Project: ws.ProjectName(),
		File:    ws.ComposeFile(),
		RunID:   ws.RunID,
		// The "test" profile is what makes a recipe's seeder service exist here and
		// nowhere else. `drillback check` never activates it.
		Profiles: []string{"test"},
		Debug:    debug,
	}

	composeUp := false
	// Teardown is registered before the first resource exists and runs on every exit
	// path, including a panic. See SPEC.md section 4.4.
	defer func() {
		if logFile != nil {
			// The log has to be closed before the directory holding it can go: on
			// Windows an open handle makes the whole teardown fail.
			_ = logFile.Close()
		}
		if o.Keep {
			kept = &Kept{Stage: "B", Workspace: ws.Root, Project: ws.ProjectName()}
			return
		}
		downCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
		defer cancel()
		if composeUp {
			if downErr := cli.Down(downCtx); downErr != nil && debug != nil {
				_, _ = fmt.Fprintf(debug, "teardown: %v\n", downErr)
			}
		}
		if rmErr := ws.Remove(); rmErr != nil && err == nil {
			err = rmErr
		}
	}()

	res, err := recipe.Resolve(o.Recipe, recipe.Options{
		InputsDir:     ws.InputsDir(),
		TestAssetsDir: ws.TestAssetsDir(),
		ExportDir:     ws.ExportDir(),
		RunID:         ws.RunID,
	})
	if err != nil {
		return failStage(st, err)
	}
	if err := emptyInputs(res); err != nil {
		return failStage(st, err)
	}
	if err := stageTestAssets(o.Recipe, ws.TestAssetsDir()); err != nil {
		return failStage(st, err)
	}

	// ---- bring the stack up ------------------------------------------------
	upStart := time.Now()
	composeRaw, err := o.Recipe.ReadFile("compose.yaml")
	if err != nil {
		return failStage(st, err)
	}
	if err := safety.Validate(composeRaw, res); err != nil {
		return failStage(st, err)
	}
	rendered, err := safety.Render(composeRaw, res.ComposeEnv())
	if err != nil {
		return failStage(st, err)
	}
	// The recipe's own mounts are checked for containment before the harness adds
	// anything, so the rule is enforced against what the author wrote. The mount
	// added next is the workspace's own export directory, by construction.
	if err := safety.CheckResolvedMounts(rendered, ws.Root); err != nil {
		return failStage(st, err)
	}
	withExport, err := withExportMount(rendered, ws.ExportDir())
	if err != nil {
		return failStage(st, err)
	}
	labelled, err := runner.LabelCompose(withExport, ws.RunID)
	if err != nil {
		return failStage(st, err)
	}
	if err := os.WriteFile(ws.ComposeFile(), labelled, 0o644); err != nil {
		return failStage(st, fmt.Errorf("writing the harness compose file: %w", err))
	}

	seedCtx, cancelSeed := context.WithTimeout(ctx, b.seed)
	defer cancelSeed()

	composeUp = true
	if _, upErr := cli.Up(seedCtx, o.Pull); upErr != nil {
		st.Phases = append(st.Phases, phaseErr("up", upStart, upErr))
		return failStage(st, upErr)
	}
	services, err := cli.Services(seedCtx)
	if err != nil {
		return failStage(st, err)
	}
	network, err := cli.NetworkName(seedCtx)
	if err != nil {
		return failStage(st, err)
	}
	st.Phases = append(st.Phases, phaseOK("up", upStart,
		fmt.Sprintf("%d services, test profile active", len(services))))

	exec := &check.Executor{
		Compose:     cli,
		Network:     network,
		HelperImage: runner.DefaultHelperImage,
		Mounts:      harnessMounts(res),
	}

	// ---- ready -------------------------------------------------------------
	readyStart := time.Now()
	probes := probe.RunAll(seedCtx, exec, res.Recipe.Ready, b.seed)
	for _, p := range probes {
		if p.Status != "ok" {
			e := fmt.Errorf("ready probe %q never succeeded after %d attempts: %s", p.Name, p.Attempts, p.Error)
			st.Phases = append(st.Phases, phaseErr("ready", readyStart, e))
			st.Logs(cli, seedCtx, services, debug)
			return failStage(st, e)
		}
	}
	st.Phases = append(st.Phases, phaseOK("ready", readyStart, probeNames(probes)))

	// ---- seed --------------------------------------------------------------
	seedStart := time.Now()
	for _, step := range res.Recipe.Test.Seed {
		if err := runStep(seedCtx, exec, step, b.seed); err != nil {
			st.Phases = append(st.Phases, phaseErr("seed", seedStart, err))
			st.Logs(cli, seedCtx, services, debug)
			return failStage(st, err)
		}
	}
	st.Phases = append(st.Phases, phaseOK("seed", seedStart,
		fmt.Sprintf("%d step%s", len(res.Recipe.Test.Seed), plural(len(res.Recipe.Test.Seed)))))

	// ---- export ------------------------------------------------------------
	exportCtx, cancelExport := context.WithTimeout(ctx, b.export)
	defer cancelExport()

	exportStart := time.Now()
	staging := filepath.Join(ws.Root, "staging")
	for _, step := range res.Recipe.Test.Export {
		if err := runStep(exportCtx, exec, step, b.export); err != nil {
			st.Phases = append(st.Phases, phaseErr("export", exportStart, err))
			st.Logs(cli, exportCtx, services, debug)
			return failStage(st, err)
		}
	}
	if err := collect(exportCtx, cli, res, ws.ExportDir(), staging); err != nil {
		st.Phases = append(st.Phases, phaseErr("export", exportStart, err))
		return failStage(st, err)
	}
	stat, _ := workspace.Measure(staging)
	st.Phases = append(st.Phases, phaseOK("export", exportStart,
		fmt.Sprintf("%d files, %d bytes staged", stat.Files, stat.Bytes)))

	// ---- back up -----------------------------------------------------------
	resticStart := time.Now()
	repoDir := filepath.Join(ws.Root, "repo")
	if err := o.backup(exportCtx, cli, repoDir, staging, res); err != nil {
		st.Phases = append(st.Phases, phaseErr("restic", resticStart, err))
		return failStage(st, err)
	}
	st.Phases = append(st.Phases, phaseOK("restic", resticStart, "throwaway repository, 1 snapshot"))

	// ---- tear the stack down before restoring ------------------------------
	// Nothing may survive in a volume: the check that follows has to see only what
	// came back out of the repository.
	downStart := time.Now()
	downCtx, cancelDown := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancelDown()
	if downErr := cli.Down(downCtx); downErr != nil {
		st.Phases = append(st.Phases, phaseErr("down", downStart, downErr))
		return failStage(st, downErr)
	}
	composeUp = false
	st.Phases = append(st.Phases, phaseOK("down", downStart, "stack and volumes removed"))

	// ---- the real command --------------------------------------------------
	// Not `check --from <repoDir>`: repoDir is inside the harness workspace, which
	// this function deletes on the way out unless --keep was passed, so that command
	// used to answer "no such file or directory" for everyone who pasted it. The one
	// below rebuilds the whole stage, which is what a contributor actually wants.
	// See ADR-061.
	st.Command = fmt.Sprintf("drillback recipe test %s --stage b --keep", recipeRef(o.Recipe))

	checkStart := time.Now()
	var rep *report.Report
	var innerKept *runner.Kept
	var runErr error
	withResticEnv(repoPassword, func() {
		rep, innerKept, runErr = runner.Run(ctx, runner.Options{
			Recipe:          o.Recipe,
			SourceKind:      "restic",
			From:            repoDir,
			Snapshot:        "latest",
			Timeout:         b.check,
			Pull:            o.Pull,
			WorkspaceParent: o.WorkspaceParent,
			Keep:            o.Keep,
			Version:         o.Version,
			Commit:          o.Commit,
			Debug:           o.Debug,
		})
	})
	if innerKept != nil {
		// Two workspaces are kept in this case: the harness's own, reported by the
		// deferred teardown, and the inner check's.
		st.Note("kept the check workspace at " + innerKept.Workspace)
	}
	if runErr != nil {
		// A tool error still produces a report - since ADR-058 it carries the stages
		// that ran and a hint - so it is worth keeping too.
		st.Check = rep
		st.Phases = append(st.Phases, phaseErr("check", checkStart, runErr))
		return failStage(st, runErr)
	}
	if rep.Verdict != report.VerdictPass {
		st.Check = rep
		st.Phases = append(st.Phases, Phase{
			Name:       "check",
			Status:     StatusFail,
			DurationMS: time.Since(checkStart).Milliseconds(),
			Note: fmt.Sprintf("%d of %d checks failed: %s",
				rep.Summary.ChecksFailed, rep.Summary.ChecksTotal, failedIDs(rep)),
		})
		st.Status = StatusFail
		st.Reason = roundTripFailure(rep)
		return st, nil, nil
	}
	st.Phases = append(st.Phases, phaseOK("check", checkStart,
		fmt.Sprintf("%d of %d checks passed", rep.Summary.ChecksPassed, rep.Summary.ChecksTotal)))
	st.Status = StatusPass
	st.Reason = fmt.Sprintf("the round trip restored and all %d checks passed", rep.Summary.ChecksTotal)
	return st, nil, nil
}

// roundTripFailure says what a stage B failure means, which is never "the tool is
// broken": the recipe seeded data, backed it up, restored it, and the checks that are
// supposed to find it did not.
func roundTripFailure(rep *report.Report) string {
	if rep.Summary.ChecksFailed == 0 {
		if rep.Error != "" {
			return "the round trip did not reach the checks: " + rep.Error
		}
		return "the round trip did not reach the checks"
	}
	return fmt.Sprintf("%d of %d checks failed after a real round trip (%s): "+
		"the seed, the export or the check disagree about where the data lives",
		rep.Summary.ChecksFailed, rep.Summary.ChecksTotal, failedIDs(rep))
}

// stageTestAssets copies the recipe's test/ directory into ${DRILLBACK_TEST_ASSETS}.
// It works for a recipe on disk and for one compiled into the binary.
func stageTestAssets(r *recipe.Recipe, dest string) error {
	src, err := r.FS()
	if err != nil {
		return err
	}
	if _, err := fs.Stat(src, "test"); err != nil {
		return nil // a recipe needs no test assets; uptime-kuma has one, gitea has none
	}
	return fs.WalkDir(src, "test", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, "test"), "/")
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := fs.ReadFile(src, p)
		if err != nil {
			return fmt.Errorf("reading test asset %s: %w", p, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

// runStep executes one seed or export step. A step is a check with a fixed
// expectation, so it goes through the same executor, and a failure is described in
// the same "expected / got" vocabulary the report already speaks.
func runStep(ctx context.Context, e *check.Executor, s *recipe.Step, budget time.Duration) error {
	c := &recipe.Check{
		ID: s.Name, Title: s.Name, Kind: s.Kind, Timeout: s.Timeout,
		Service: s.Service, User: s.User, Command: s.Command,
		URL: s.URL, Method: s.Method, BasicAuth: s.BasicAuth, JSONBody: s.JSONBody,
	}
	switch s.Kind {
	case "exec":
		zero := 0
		c.Expect = recipe.Expect{ExitCode: &zero}
	case "http":
		want := s.ExpectStatus
		if want == 0 {
			want = 200
		}
		c.Expect = recipe.Expect{Status: &want}
	default:
		return fmt.Errorf("step %q: unknown kind %q", s.Name, s.Kind)
	}
	r := check.Run(ctx, e, c, budget)
	if r.Passed() {
		return nil
	}
	return fmt.Errorf("step %q: %s", s.Name, stepFailure(r))
}

func stepFailure(r check.Result) string {
	if r.Observed.Error != "" {
		return r.Observed.Error
	}
	var parts []string
	for _, f := range r.Failures {
		parts = append(parts, fmt.Sprintf("expected %s, got %s", f.Expect, f.Got))
	}
	if out := lastLine(r.Observed.Stderr); out != "" {
		parts = append(parts, "stderr: "+out)
	} else if out := lastLine(r.Observed.Stdout); out != "" {
		parts = append(parts, "stdout: "+out)
	} else if out := lastLine(r.Observed.Body); out != "" {
		parts = append(parts, "body: "+out)
	}
	return strings.Join(parts, "; ")
}

// collect assembles the staging tree: the tree a backup of this application would
// have contained, laid out at the paths the recipe declares.
func collect(ctx context.Context, cli *compose.Client, res *recipe.Resolved, exportDir, staging string) error {
	for _, in := range res.Inputs {
		if in.Within != "" {
			continue // it lives inside its parent and travels with it
		}
		dest := filepath.Join(staging, filepath.FromSlash(strings.TrimPrefix(in.BackupPath, "/")))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if in.Kind == "dir" {
			if err := exportDirInput(ctx, cli, in, dest); err != nil {
				return err
			}
			continue
		}
		// A non-dir input has to be produced by an export step, because only the
		// application knows how to serialise its own database.
		produced := filepath.Join(exportDir, path.Base(in.BackupPath))
		if _, err := os.Stat(produced); err != nil {
			return fmt.Errorf("no export step produced input %q: a step with `produces: %s` must "+
				"leave its artifact at $DRILLBACK_EXPORT/%s", in.Name, in.Name, path.Base(in.BackupPath))
		}
		if err := workspace.CopyTree(produced, dest); err != nil {
			return fmt.Errorf("staging input %q: %w", in.Name, err)
		}
	}
	return nil
}

// exportDirInput copies a dir input out of the container that owns it. The daemon
// reads the files, so a tree the application wrote as its own user is staged without
// the harness having to be that user.
func exportDirInput(ctx context.Context, cli *compose.Client, in *recipe.ResolvedInput, dest string) error {
	if in.Mount == nil {
		// Nothing mounted it, so the workspace copy is the only copy there is.
		return workspace.CopyTree(in.LocalPath, dest)
	}
	service, containerPath, ok := strings.Cut(in.Mount.Into, ":")
	if !ok {
		return fmt.Errorf("input %q: mount.into %q is not service:path", in.Name, in.Mount.Into)
	}
	if err := cli.CopyOut(ctx, service, containerPath, dest); err != nil {
		return fmt.Errorf("exporting input %q from %s:%s: %w", in.Name, service, containerPath, err)
	}
	return nil
}

// backup creates the throwaway repository and puts the staging tree into it.
//
// It runs restic in a container rather than on the host, with each staged input bound
// at the absolute path the recipe declares for it. That is what makes the snapshot
// record /srv/gitea/data rather than a path inside the workspace, and it is what lets
// step 8 be the command a user actually types. See DECISIONS.md ADR-051.
func (o Options) backup(ctx context.Context, cli *compose.Client, repoDir, staging string, res *recipe.Resolved) error {
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return fmt.Errorf("creating the throwaway repository directory: %w", err)
	}
	env := map[string]string{"RESTIC_PASSWORD": repoPassword}
	base := []compose.Bind{{Host: dockerPath(repoDir), Container: "/repo"}}

	init, err := cli.RunContainer(ctx, compose.ContainerOptions{
		Image: o.ResticImage,
		Env:   env,
		Binds: base,
		User:  containerUser(),
		Argv:  []string{"--no-cache", "--repo", "/repo", "init", "--repository-version", "2"},
	})
	if err != nil {
		return fmt.Errorf("running restic in %s: %w", o.ResticImage, err)
	}
	if init.ExitCode != 0 {
		return fmt.Errorf("restic init failed: %s", lastLine(init.Combined()))
	}

	binds := base
	var paths []string
	for _, in := range res.Inputs {
		if in.Within != "" {
			continue
		}
		host := filepath.Join(staging, filepath.FromSlash(strings.TrimPrefix(in.BackupPath, "/")))
		binds = append(binds, compose.Bind{Host: dockerPath(host), Container: in.BackupPath, ReadOnly: true})
		paths = append(paths, in.BackupPath)
	}
	if len(paths) == 0 {
		return errors.New("the recipe declares no inputs, so there is nothing to back up")
	}

	argv := append([]string{"--no-cache", "--repo", "/repo", "backup",
		"--tag", repoTag, "--host", repoHost}, paths...)
	out, err := cli.RunContainer(ctx, compose.ContainerOptions{
		Image: o.ResticImage,
		Env:   env,
		Binds: binds,
		User:  containerUser(),
		Argv:  argv,
	})
	if err != nil {
		return fmt.Errorf("running restic in %s: %w", o.ResticImage, err)
	}
	if out.ExitCode != 0 {
		return fmt.Errorf("restic backup failed: %s", lastLine(out.Combined()))
	}
	return nil
}

// containerUser is the uid:gid the restic container runs as. On Linux the repository
// has to belong to the caller, because the `drillback check` that follows runs as the
// caller and restic takes a lock. On Windows the daemon does not map ownership and
// the flag is not meaningful.
func containerUser() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
}

// dockerPath renders a host path the way the docker CLI wants to see it in a -v
// argument. On Windows a backslash path is ambiguous with the volume separator, and
// the forward-slash form of the same absolute path is accepted.
func dockerPath(p string) string { return filepath.ToSlash(p) }

// withResticEnv runs fn with the throwaway repository's password in the environment.
//
// restic reads its password from the environment, and a user testing a recipe very
// likely has RESTIC_PASSWORD_FILE or RESTIC_PASSWORD_COMMAND pointing at their own
// repository. Both take precedence over RESTIC_PASSWORD, so both are cleared for the
// duration of the call and put back afterwards.
func withResticEnv(password string, fn func()) {
	type saved struct {
		value string
		set   bool
	}
	names := []string{"RESTIC_PASSWORD", "RESTIC_PASSWORD_FILE", "RESTIC_PASSWORD_COMMAND", "RESTIC_REPOSITORY_FILE"}
	old := make(map[string]saved, len(names))
	for _, n := range names {
		v, ok := os.LookupEnv(n)
		old[n] = saved{v, ok}
		_ = os.Unsetenv(n)
	}
	_ = os.Setenv("RESTIC_PASSWORD", password)
	defer func() {
		for _, n := range names {
			if s := old[n]; s.set {
				_ = os.Setenv(n, s.value)
			} else {
				_ = os.Unsetenv(n)
			}
		}
	}()
	fn()
}

func harnessMounts(res *recipe.Resolved) []check.Mount {
	var out []check.Mount
	for _, in := range res.Inputs {
		if in.Mount == nil {
			continue
		}
		svc, p, ok := strings.Cut(in.Mount.Into, ":")
		if !ok {
			continue
		}
		out = append(out, check.Mount{Service: svc, ContainerPath: p, HostPath: in.LocalPath})
	}
	return out
}

func failStage(st Stage, err error) (Stage, *Kept, error) {
	st.Status = StatusError
	st.Error = err.Error()
	return st, nil, err
}

func phaseOK(name string, start time.Time, note string) Phase {
	return Phase{Name: name, Status: StatusPass, DurationMS: time.Since(start).Milliseconds(), Note: note}
}

func phaseErr(name string, start time.Time, err error) Phase {
	return Phase{Name: name, Status: StatusError, DurationMS: time.Since(start).Milliseconds(), Error: err.Error()}
}

// Note appends a line to the stage's reason, for facts a human needs that are not the
// verdict itself.
func (s *Stage) Note(note string) {
	if s.Reason == "" {
		s.Reason = note
		return
	}
	s.Reason += "; " + note
}

// Logs writes the services' logs into the harness log, which is where a contributor
// looks when a seed step failed for a reason the step's own output does not explain.
func (s *Stage) Logs(cli *compose.Client, ctx context.Context, services []string, debug io.Writer) {
	if debug == nil {
		return
	}
	for _, svc := range services {
		logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		text, err := cli.Logs(logCtx, svc, 200)
		cancel()
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		_, _ = fmt.Fprintf(debug, "\n--- logs: %s ---\n%s\n", svc, text)
	}
}

func probeNames(ps []probe.Result) string {
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		names = append(names, p.Name)
	}
	return strings.Join(names, ", ")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}
	return ""
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
