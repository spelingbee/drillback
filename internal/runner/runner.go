// Package runner is the run lifecycle: the eight-state machine of SPEC.md section 4.
// It owns the order, the budgets, and the guarantee that everything it created is
// destroyed again.
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spelingbee/restored/internal/check"
	"github.com/spelingbee/restored/internal/compose"
	"github.com/spelingbee/restored/internal/hints"
	"github.com/spelingbee/restored/internal/loader"
	"github.com/spelingbee/restored/internal/probe"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/recipe/safety"
	"github.com/spelingbee/restored/internal/report"
	"github.com/spelingbee/restored/internal/source"
	dirsource "github.com/spelingbee/restored/internal/source/dir"
	resticsource "github.com/spelingbee/restored/internal/source/restic"
	"github.com/spelingbee/restored/internal/workspace"
)

// DefaultHelperImage is the throwaway container every HTTP and TCP check runs from.
// It is pinned, and it is the only image restored itself introduces into a run.
const DefaultHelperImage = "curlimages/curl:8.16.0"

// Options is one invocation of `restored check`.
type Options struct {
	Recipe *recipe.Recipe

	SourceKind string
	From       string
	Snapshot   string
	Tags       []string
	Host       string

	InputPaths map[string]string
	SetVars    map[string]string

	Timeout        time.Duration
	RestoreTimeout time.Duration
	ReadyTimeout   time.Duration
	CheckTimeout   time.Duration

	Pull            string
	WorkspaceParent string
	Keep            bool
	KeepOnFail      bool
	HintsFile       string

	Version string
	Commit  string
	Debug   io.Writer
}

// Kept describes a workspace and compose project the run deliberately left behind.
type Kept struct {
	Workspace string
	Project   string
}

// Run executes the whole lifecycle. A non-nil error is a tool error and maps to exit
// 2; a report with verdict RESTORE_UNUSABLE maps to exit 1.
func Run(ctx context.Context, o Options) (rep *report.Report, kept *Kept, err error) {
	started := time.Now()
	o.applyDefaults()

	rep = &report.Report{
		SchemaVersion: report.SchemaVersion,
		Tool:          report.Tool{Name: "restored", Version: o.Version, Commit: o.Commit},
		Verdict:       report.VerdictError,
		ExitCode:      2,
		Recipe: report.RecipeInfo{
			Name:   o.Recipe.Metadata.Name,
			Title:  o.Recipe.Metadata.Title,
			Source: recipeOrigin(o.Recipe),
			Digest: o.Recipe.Digest,
		},
	}

	// A tool error is still a run that happened, and the user still needs to be told
	// what to do about it. Before this, attachHint was reached only from the two
	// exit-1 paths, so every rule in the catalog about a tool error - an unreachable
	// docker daemon, a failed image pull, a path that is not in the snapshot, a full
	// disk, a wrong restic password - was unreachable code. See ADR-058 and
	// docs/review/architecture.md ARCH-04.
	//
	// Registered first, so it runs last: teardown has finished by then and the logs
	// it collected are in the report the hint is matched against.
	var resolved *recipe.Resolved
	defer func() {
		if err == nil || rep == nil {
			return
		}
		if rep.Error == "" {
			rep.Error = err.Error()
		}
		rep.Verdict = report.VerdictError
		rep.ExitCode = 2
		o.attachHint(rep, resolved)
		finish(rep, started)
	}()

	// The run budget is the outer bound from the first syscall, not from after the
	// preflight. Preflight shells out to docker, and a docker that is not answering
	// is exactly the case a --timeout is set for. See ARCH-05.
	runCtx, cancelAll := context.WithTimeout(ctx, o.Timeout)
	defer cancelAll()

	// ---- RESOLVE -----------------------------------------------------------
	if err := compose.Preflight(runCtx, o.SourceKind == "restic"); err != nil {
		return rep, nil, err
	}

	// ---- PREPARE -----------------------------------------------------------
	ws, err := workspace.New(o.WorkspaceParent)
	if err != nil {
		return rep, nil, err
	}
	rep.Run = report.Run{
		ID:             ws.RunID,
		ComposeProject: ws.ProjectName(),
		StartedAt:      started.UTC().Format(time.RFC3339),
		Workspace:      ws.Root,
	}

	debug := o.Debug
	logFile, logErr := os.Create(filepath.Join(ws.LogsDir(), "debug.log"))
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
		Debug:   debug,
	}

	// Teardown is registered before the first resource exists, and runs on every
	// exit path including a panic. See SPEC.md section 4.4.
	composeUp := false
	defer func() {
		keepIt := o.Keep || (o.KeepOnFail && (err != nil || rep.Verdict != report.VerdictPass))
		if logFile != nil {
			// The debug log has to be closed before the directory holding it can be
			// removed. On Windows an open handle makes the whole teardown fail.
			_ = logFile.Close()
		}
		if keepIt {
			kept = &Kept{Workspace: ws.Root, Project: ws.ProjectName()}
			rep.Run.WorkspaceRemoved = false
			return
		}
		downCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
		defer cancel()
		if composeUp {
			if downErr := cli.Down(downCtx); downErr != nil && debug != nil {
				_, _ = fmt.Fprintf(debug, "teardown: %v\n", downErr)
			}
		}
		if rmErr := ws.Remove(); rmErr != nil {
			rep.Run.WorkspaceRemoved = false
			if err == nil {
				err = rmErr
			}
			return
		}
		rep.Run.WorkspaceRemoved = true
	}()

	res, err := recipe.Resolve(o.Recipe, recipe.Options{
		InputPaths:    o.InputPaths,
		Vars:          o.SetVars,
		InputsDir:     ws.InputsDir(),
		TestAssetsDir: ws.TestAssetsDir(),
		ExportDir:     ws.ExportDir(),
		RunID:         ws.RunID,
	})
	if err != nil {
		return rep, kept, err
	}
	// The deferred finaliser matches hints against the recipe's inputs, so it needs
	// the resolution as soon as there is one.
	resolved = res

	composeRaw, err := o.Recipe.ReadFile("compose.yaml")
	if err != nil {
		return rep, kept, err
	}
	if err := safety.Validate(composeRaw, res); err != nil {
		return rep, kept, err
	}

	// ---- RESTORE -----------------------------------------------------------
	restoreStart := time.Now()
	desc, warnings, err := o.materialise(runCtx, ws, res)
	if err != nil {
		rep.Stages = append(rep.Stages, report.Stage{
			Name: "restore", Status: "failed",
			DurationMS: time.Since(restoreStart).Milliseconds(), Error: err.Error(),
		})
		return rep, kept, err
	}
	rep.Source = desc
	rep.Warnings = warnings
	rep.Inputs = inputReports(res)
	rep.Stages = append(rep.Stages, report.Stage{
		Name: "restore", Status: "ok",
		DurationMS: time.Since(restoreStart).Milliseconds(),
		Note:       fmt.Sprintf("%d input%s", len(res.Inputs), plural(len(res.Inputs))),
	})

	// ---- COMPOSE UP --------------------------------------------------------
	upStart := time.Now()
	rendered, err := safety.Render(composeRaw, res.ComposeEnv())
	if err != nil {
		return rep, kept, err
	}
	if err := safety.CheckResolvedMounts(rendered, ws.Root); err != nil {
		return rep, kept, err
	}
	labelled, err := LabelCompose(rendered, ws.RunID)
	if err != nil {
		return rep, kept, err
	}
	if err := os.WriteFile(ws.ComposeFile(), labelled, 0o644); err != nil {
		return rep, kept, fmt.Errorf("writing the run's compose file: %w", err)
	}

	// A dump has to be in the database before the application first connects to it:
	// an application that starts against an empty database runs its own migrations,
	// and the dump then collides with the schema it just created. So the services
	// that receive a dump start first, and the rest start once the load is done.
	// See DECISIONS.md ADR-041.
	loadFirst := loadServices(res)
	composeUp = true
	if _, upErr := cli.Up(runCtx, o.Pull, loadFirst...); upErr != nil {
		rep.Stages = append(rep.Stages, report.Stage{
			Name: "compose", Status: "failed",
			DurationMS: time.Since(upStart).Milliseconds(), Error: upErr.Error(),
		})
		rep.Logs = collectLogs(runCtx, cli, nil)
		return rep, kept, upErr
	}
	services, err := cli.Services(runCtx)
	if err != nil {
		return rep, kept, err
	}
	network, err := cli.NetworkName(runCtx)
	if err != nil {
		return rep, kept, err
	}
	composeNote := fmt.Sprintf("%d service%s", len(services), plural(len(services)))
	if len(loadFirst) > 0 {
		composeNote = fmt.Sprintf("%d service%s, %s first for the dump",
			len(services), plural(len(services)), strings.Join(loadFirst, ", "))
	}
	rep.Stages = append(rep.Stages, report.Stage{
		Name: "compose", Status: "ok",
		DurationMS: time.Since(upStart).Milliseconds(),
		Services:   services,
		Note:       composeNote,
	})

	exec := &check.Executor{
		Compose:     cli,
		Network:     network,
		HelperImage: helperImage(),
		Mounts:      mountsOf(res),
	}

	// From here on, a failure is a verdict about the backup, not a tool error.
	// See SPEC.md section 4.2.
	fail := func(stage string, e error) (*report.Report, *Kept, error) {
		rep.Verdict = report.VerdictUnusable
		rep.ExitCode = 1
		rep.Error = e.Error()
		// Unless the reason is that restored ran out of time. A cancelled runCtx
		// looks exactly like a failed stage from in here - the probe records
		// `context deadline exceeded` and the stage is marked failed - and calling
		// that an unusable restore accuses a backup that may be perfectly good. The
		// cause decides the verdict, not the stage. See DECISIONS.md ADR-058.
		if ctxErr := runCtx.Err(); ctxErr != nil {
			rep.Verdict = report.VerdictError
			rep.ExitCode = 2
			switch {
			case errors.Is(ctxErr, context.DeadlineExceeded):
				rep.Error = fmt.Sprintf(
					"restored ran out of its --timeout budget of %s during the %s stage. "+
						"Nothing is known about the backup: the drill did not finish. "+
						"Re-run with a longer --timeout.", o.Timeout, stage)
			default:
				rep.Error = fmt.Sprintf("the run was cancelled during the %s stage. "+
					"Nothing is known about the backup: the drill did not finish.", stage)
			}
		}
		rep.Logs = collectLogs(runCtx, cli, services)
		rep.Summary = report.Summary{ChecksTotal: len(res.Recipe.Checks), ChecksSkipped: len(res.Recipe.Checks)}
		o.attachHint(rep, res)
		finish(rep, started)
		return rep, kept, nil
	}

	// ---- LOAD DUMPS --------------------------------------------------------
	loadStart := time.Now()
	detail := map[string]any{}
	for _, in := range res.Inputs {
		switch in.Kind {
		case "postgres-dump":
			d, loadErr := loader.LoadPostgres(runCtx, cli, in, loadTimeout(in))
			detail[in.Name] = d
			if loadErr != nil {
				rep.Stages = append(rep.Stages, report.Stage{
					Name: "load db", Status: "failed",
					DurationMS: time.Since(loadStart).Milliseconds(),
					Detail:     detail, Error: loadErr.Error(),
				})
				return fail("load_dumps", loadErr)
			}
		case "sqlite":
			if in.Load == nil || !in.Load.IntegrityCheck {
				continue
			}
			d, loadErr := loader.IntegrityCheck(runCtx, in)
			detail[in.Name] = d
			if loadErr != nil {
				rep.Stages = append(rep.Stages, report.Stage{
					Name: "load db", Status: "failed",
					DurationMS: time.Since(loadStart).Milliseconds(),
					Detail:     detail, Error: loadErr.Error(),
				})
				return fail("load_dumps", loadErr)
			}
		}
	}
	if len(detail) > 0 {
		rep.Stages = append(rep.Stages, report.Stage{
			Name: "load db", Status: "ok",
			DurationMS: time.Since(loadStart).Milliseconds(),
			Detail:     detail, Note: loadNote(detail),
		})
	}

	// ---- READY -------------------------------------------------------------
	// The application starts here, against a database that already holds the restore.
	readyStart := time.Now()
	if len(loadFirst) > 0 {
		if _, upErr := cli.Up(runCtx, o.Pull); upErr != nil {
			rep.Stages = append(rep.Stages, report.Stage{
				Name: "ready", Status: "failed",
				DurationMS: time.Since(readyStart).Milliseconds(), Error: upErr.Error(),
			})
			return fail("ready", upErr)
		}
	}
	probes := probe.RunAll(runCtx, exec, res.Recipe.Ready, o.ReadyTimeout)
	stage := report.Stage{
		Name: "ready", Status: "ok",
		DurationMS: time.Since(readyStart).Milliseconds(),
		Probes:     probeReports(probes),
		Note:       probeNote(probes),
	}
	for _, p := range probes {
		if p.Status != "ok" {
			stage.Status = "failed"
			stage.Error = fmt.Sprintf("%s: %s", p.Name, p.Error)
		}
	}
	rep.Stages = append(rep.Stages, stage)
	if stage.Status != "ok" {
		return fail("ready", errors.New(stage.Error))
	}

	// ---- CHECKS ------------------------------------------------------------
	for _, c := range res.Recipe.Checks {
		r := check.Run(runCtx, exec, c, o.CheckTimeout)
		rep.Checks = append(rep.Checks, report.Check{
			ID: c.ID, Title: c.Title, Kind: c.Kind,
			Status: r.Status, DurationMS: r.Duration.Milliseconds(),
			Expect: c.Expect, Observed: r.Observed,
			Query: c.Query, URL: c.URL,
			Failures: r.Failures,
		})
		rep.Summary.ChecksTotal++
		if r.Passed() {
			rep.Summary.ChecksPassed++
		} else {
			rep.Summary.ChecksFailed++
		}
	}

	// ---- REPORT ------------------------------------------------------------
	if rep.Summary.ChecksFailed > 0 {
		rep.Verdict = report.VerdictUnusable
		rep.ExitCode = 1
		rep.Logs = collectLogs(runCtx, cli, services)
		o.attachHint(rep, res)
	} else {
		rep.Verdict = report.VerdictPass
		rep.ExitCode = 0
	}
	finish(rep, started)
	return rep, kept, nil
}

func finish(rep *report.Report, started time.Time) {
	now := time.Now()
	rep.Run.FinishedAt = now.UTC().Format(time.RFC3339)
	rep.Run.DurationMS = now.Sub(started).Milliseconds()
}

func (o *Options) applyDefaults() {
	// 30 minutes, not 15. The stage budgets below already sum to 15, so a 15-minute
	// run budget guaranteed that a big restore left nothing for the stages after it
	// and the run died of its own defaults. SPEC.md 4.2's per-state budgets sum to
	// roughly 27 minutes. See DECISIONS.md ADR-058.
	if o.Timeout <= 0 {
		o.Timeout = 30 * time.Minute
	}
	if o.RestoreTimeout <= 0 {
		o.RestoreTimeout = 10 * time.Minute
	}
	if o.ReadyTimeout <= 0 {
		o.ReadyTimeout = 5 * time.Minute
	}
	if o.CheckTimeout <= 0 {
		o.CheckTimeout = 60 * time.Second
	}
	// A stage budget is the ceiling for one stage; --timeout is the ceiling for all
	// of them together. When a user shortens --timeout, the stage budgets have to
	// come down with it, or the first stage consumes the whole run and every stage
	// after it is judged against a deadline it never had a chance against. Clamp
	// down, never up: an explicit --restore-timeout larger than the run is a
	// contradiction, and the run is the one the user typed last.
	if limit := o.Timeout / 2; o.RestoreTimeout > limit {
		o.RestoreTimeout = limit
	}
	if limit := o.Timeout / 4; o.ReadyTimeout > limit {
		o.ReadyTimeout = limit
	}
	if limit := o.Timeout / 8; o.CheckTimeout > limit {
		o.CheckTimeout = limit
	}
	if o.Pull == "" {
		o.Pull = "missing"
	}
	if o.SourceKind == "" {
		o.SourceKind = "restic"
	}
	if o.Version == "" {
		o.Version = "0.0.0-dev"
	}
}

func helperImage() string {
	if v := os.Getenv("RESTORED_HELPER_IMAGE"); v != "" {
		return v
	}
	return DefaultHelperImage
}

func recipeOrigin(r *recipe.Recipe) string {
	if r.Bundled {
		return "bundled"
	}
	return "path"
}

func loadTimeout(in *recipe.ResolvedInput) time.Duration {
	if in.Load == nil || in.Load.Timeout == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(in.Load.Timeout)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}

// loadServices lists the services that must be running before the dumps can be
// loaded, in recipe order and without repeats.
func loadServices(res *recipe.Resolved) []string {
	var out []string
	seen := map[string]bool{}
	for _, in := range res.Inputs {
		if in.Kind != "postgres-dump" || in.Load == nil || in.Load.Service == "" {
			continue
		}
		if !seen[in.Load.Service] {
			seen[in.Load.Service] = true
			out = append(out, in.Load.Service)
		}
	}
	return out
}

func mountsOf(res *recipe.Resolved) []check.Mount {
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

func inputReports(res *recipe.Resolved) []report.Input {
	out := make([]report.Input, 0, len(res.Inputs))
	for _, in := range res.Inputs {
		st, _ := workspace.Measure(in.LocalPath)
		ri := report.Input{
			Name: in.Name, Kind: in.Kind, BackupPath: in.BackupPath,
			Bytes: st.Bytes, Files: st.Files, Origin: string(in.Origin),
		}
		if in.Kind == "postgres-dump" {
			if f, err := loader.DetectFormat(in.LocalPath); err == nil {
				ri.DetectedFormat = f
			}
		}
		out = append(out, ri)
	}
	return out
}

func probeReports(ps []probe.Result) []report.ProbeResult {
	out := make([]report.ProbeResult, 0, len(ps))
	for _, p := range ps {
		out = append(out, report.ProbeResult{
			Name: p.Name, Status: p.Status, Attempts: p.Attempts,
			DurationMS: p.DurationMS, Error: p.Error,
		})
	}
	return out
}

func probeNote(ps []probe.Result) string {
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		if p.Status == "ok" {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ", ")
}

func loadNote(detail map[string]any) string {
	var notes []string
	for name, d := range detail {
		if dd, ok := d.(loader.Detail); ok {
			notes = append(notes, fmt.Sprintf("%s: %s, %d stderr line%s",
				name, dd.Loader, dd.StderrLines, plural(dd.StderrLines)))
		}
	}
	return strings.Join(notes, "; ")
}

func collectLogs(ctx context.Context, cli *compose.Client, services []string) map[string][]string {
	if len(services) == 0 {
		var err error
		services, err = cli.Services(ctx)
		if err != nil {
			return nil
		}
	}
	out := map[string][]string{}
	for _, s := range services {
		logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		text, err := cli.Logs(logCtx, s, 200)
		cancel()
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		out[s] = strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// materialise restores the recipe's inputs into the workspace and sanitises them.
func (o *Options) materialise(ctx context.Context, ws *workspace.Workspace, res *recipe.Resolved) (source.Descriptor, []report.Warning, error) {
	restoreCtx, cancel := context.WithTimeout(ctx, o.RestoreTimeout)
	defer cancel()

	reqs := make([]source.Request, 0, len(res.Inputs))
	for _, in := range res.Inputs {
		if in.Within != "" {
			continue
		}
		reqs = append(reqs, source.Request{
			Name: in.Name, BackupPath: in.BackupPath, LocalPath: in.LocalPath, Required: in.Required,
		})
	}

	var desc source.Descriptor
	var locate func(string) string

	switch o.SourceKind {
	case "restic":
		opts := resticsource.Options{
			Repository: o.From, Snapshot: o.Snapshot, Tags: o.Tags, Host: o.Host, Debug: o.Debug,
		}
		snaps, err := resticsource.ListSnapshots(restoreCtx, opts)
		if err != nil {
			return desc, nil, err
		}
		snap, err := resticsource.Select(snaps, o.Snapshot)
		if err != nil {
			return desc, nil, err
		}
		if err := resticsource.Restore(restoreCtx, opts, snap, reqs, ws.RestoreDir()); err != nil {
			return desc, nil, err
		}
		desc = source.Descriptor{Kind: "restic", Repository: repositoryLabel(o.From), Snapshot: snap}
		locate = func(p string) string { return resticsource.Locate(ws.RestoreDir(), p) }
	case "dir":
		if o.From == "" {
			return desc, nil, errors.New("--source dir needs --from <tree>")
		}
		if err := dirsource.Check(o.From); err != nil {
			return desc, nil, err
		}
		abs, err := filepath.Abs(o.From)
		if err != nil {
			return desc, nil, err
		}
		desc = source.Descriptor{Kind: "dir", Repository: abs}
		locate = func(p string) string { return dirsource.Locate(abs, p) }
	default:
		return desc, nil, fmt.Errorf("unknown source %q: restored reads restic or dir", o.SourceKind)
	}

	var warnings []report.Warning
	for _, in := range res.Inputs {
		if in.Within != "" {
			if _, err := os.Stat(in.LocalPath); err != nil {
				if in.Required {
					return desc, warnings, fmt.Errorf("input %q: %s is not inside the restored %q input",
						in.Name, in.BackupPath, in.Within)
				}
			}
			continue
		}
		src := locate(in.BackupPath)
		info, err := os.Stat(src)
		if err != nil {
			if in.Required {
				// The highest-traffic error in the tool: a recipe's default paths
				// are the recipe author's layout, and most people's differ. Naming
				// the flag that fixes it, and the command that lists what the recipe
				// wants, is the difference between a second command and a dead end.
				// See docs/review/ux.md UX-03.
				return desc, warnings, fmt.Errorf(
					"required input %q: no matching files found for %s in the backup.\n"+
						"  A recipe default path is a guess at your layout. Point this input\n"+
						"  at the path your backup actually uses:\n"+
						"      --input %s=/your/path\n"+
						"  `restored recipe show %s --inputs-only` lists every input this recipe wants",
					in.Name, in.BackupPath, in.Name, res.Recipe.Metadata.Name)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(in.LocalPath), 0o755); err != nil {
			return desc, warnings, err
		}
		if o.SourceKind == "restic" {
			// The restore directory is the workspace's own, so a move costs nothing
			// and leaves one copy of the data on disk instead of two.
			if err := os.Rename(src, in.LocalPath); err != nil {
				if err := workspace.CopyTree(src, in.LocalPath); err != nil {
					return desc, warnings, err
				}
			}
		} else if err := workspace.CopyTree(src, in.LocalPath); err != nil {
			return desc, warnings, err
		}
		ws2, err := ws.Sanitise(in.LocalPath)
		if err != nil {
			return desc, warnings, err
		}
		// The uid a backup records is the original host's, and the application is
		// about to start as whatever uid its image chose. See workspace.Relax.
		if err := ws.Relax(in.LocalPath); err != nil {
			return desc, warnings, err
		}
		for _, w := range ws2 {
			warnings = append(warnings, report.Warning{Code: w.Code, Detail: w.Detail})
		}
		st, err := workspace.Measure(in.LocalPath)
		if err != nil {
			return desc, warnings, err
		}
		if st.Bytes == 0 && info.IsDir() {
			warnings = append(warnings, report.Warning{
				Code:   "empty_input",
				Detail: fmt.Sprintf("restored input %q is empty", in.Name),
			})
		}
	}
	if err := os.RemoveAll(ws.RestoreDir()); err != nil {
		return desc, warnings, err
	}
	return desc, warnings, nil
}

// repositoryLabel is what the report shows for the repository. A repository string can
// carry a user name but never a password, and restored never reads the environment
// variables that do.
// repositoryLabel is what the report shows for the repository, with any password
// taken out of it first. A restic repository string can carry one - `rest:` and every
// object-store backend accept `user:password@host` - and this string is printed on
// the terminal, serialised into --json, written by --report, and attached to bug
// reports by the issue templates. See DECISIONS.md ADR-059.
func repositoryLabel(from string) string {
	if from != "" {
		return resticsource.SafeRepository(from)
	}
	return resticsource.SafeRepository(os.Getenv("RESTIC_REPOSITORY"))
}

// attachHint matches the catalog against what the run produced. At most one hint is
// ever shown, and it never changes the verdict.
// attachHint tolerates a nil resolution: the earliest failures - an unreachable
// docker daemon, a workspace that cannot be created - happen before the recipe has
// been resolved, and those are exactly the ones the catalog has the most to say
// about.
func (o *Options) attachHint(rep *report.Report, res *recipe.Resolved) {
	catalog, err := hints.Builtin()
	if err != nil {
		return
	}
	if res == nil {
		res = &recipe.Resolved{Recipe: o.Recipe}
	}
	if o.HintsFile != "" {
		extra, err := hints.Load(o.HintsFile)
		if err == nil {
			catalog = hints.Concat(extra, catalog)
		}
	}

	var subjects []hints.Subject
	for i, c := range rep.Checks {
		if c.Status == "pass" {
			continue
		}
		driver := ""
		for _, rc := range res.Recipe.Checks {
			if rc.ID == c.ID {
				driver = rc.Driver
			}
		}
		where := fmt.Sprintf("checks[%d].observed", i)
		subjects = append(subjects,
			hints.Subject{Where: where + ".error", Text: c.Observed.Error, Driver: driver},
			hints.Subject{Where: where + ".body", Text: c.Observed.Body, Driver: driver},
			hints.Subject{Where: where + ".stderr", Text: c.Observed.Stderr, Driver: driver},
			// A check can fail without anything going wrong: the query ran, the app
			// answered, and the answer was not the one the recipe wanted. That is the
			// most common shape of an unusable restore, so the expectation and the
			// observation are offered to the catalog as well.
			hints.Subject{
				Where:  fmt.Sprintf("checks[%d].failures", i),
				Text:   failureText(c.Failures),
				Driver: driver,
			},
		)
	}
	if rep.Error != "" {
		subjects = append(subjects, hints.Subject{Where: "error", Text: rep.Error, Driver: dumpDriver(res)})
	}
	for _, w := range rep.Warnings {
		subjects = append(subjects, hints.Subject{Where: "warnings", Text: w.Detail})
	}
	for _, name := range sortedKeys(rep.Logs) {
		subjects = append(subjects, hints.Subject{
			Where: "logs." + name,
			Text:  strings.Join(rep.Logs[name], "\n"),
		})
	}

	rule, where, ok := catalog.Match(subjects)
	if !ok {
		return
	}
	inputs := map[string]string{}
	for _, in := range res.Inputs {
		inputs[in.Name] = in.BackupPath
	}
	snapID := ""
	if rep.Source.Snapshot != nil {
		snapID = rep.Source.Snapshot.ShortID
	}
	rep.Hint = &report.Hint{
		ID:        rule.ID,
		MatchedOn: where,
		Title:     rule.Title,
		Text:      rule.Text,
		Commands:  rule.RenderCommands(hints.CommandContext{Inputs: inputs, SnapshotID: snapID}),
	}
}

// failureText renders a check's unmet expectations in the stable phrasing the hint
// catalog matches against.
func failureText(failures []check.Failure) string {
	var b strings.Builder
	for _, f := range failures {
		fmt.Fprintf(&b, "expected %s, got %s\n", f.Expect, f.Got)
	}
	return b.String()
}

// dumpDriver reports which database driver a run is about, so a load failure can be
// scoped the same way a check failure is.
func dumpDriver(res *recipe.Resolved) string {
	for _, in := range res.Inputs {
		switch in.Kind {
		case "postgres-dump":
			return "postgres"
		case "sqlite":
			return "sqlite"
		}
	}
	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
