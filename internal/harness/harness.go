// Package harness runs the round-trip harness of SPEC.md section 7: the two-stage,
// mechanical proof that a recipe's checks depend on the data a restore puts back.
//
// A recipe is a claim: if these inputs are restored, these checks pass, and if they
// are not, they fail. Stage A proves the second half and stage B proves the first, so
// accepting a recipe needs no maintainer judgement about whether its checks mean
// anything.
package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spelingbee/restored/internal/compose"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/report"
	"github.com/spelingbee/restored/internal/runner"
	resticsource "github.com/spelingbee/restored/internal/source/restic"
)

// Fixed properties of the throwaway restic repository. None of them is configurable,
// because a recipe author should never have to think about restic in order to
// contribute a recipe. The password is not a secret: the repository is created inside
// the run workspace and is destroyed with it. See SPEC.md section 7.3.
const (
	repoPassword = "restored-recipe-test"
	repoTag      = "restored-recipe-test"
	repoHost     = "restored-harness"
)

// DefaultResticImage drives the throwaway repository. It is pinned, and it is used
// instead of the host's restic for the backup so that the snapshot records exactly
// the paths the recipe declares. See DECISIONS.md ADR-051.
const DefaultResticImage = "restic/restic:0.19.1"

// DefaultTimeout is the wall-clock budget for one recipe across both stages.
const DefaultTimeout = 20 * time.Minute

// stageAReady is the reduced ready budget of SPEC.md section 7.2. An application that
// will not start against no data at all is evidence, not a failure, and there is no
// reason to wait the full budget to find that out.
const stageAReady = 90 * time.Second

// NoDataSensitiveCheck is the rejection a recipe earns when every one of its checks
// passes against an empty stack. It names what is missing rather than what went
// wrong, because the author's next action is to add a check, not to debug one.
const NoDataSensitiveCheck = "recipe has no data-sensitive check: add a check that depends on restored data"

// NoCountingCheck is the rejection a recipe earns when the empty stack refused to
// start - so no check was exercised against emptiness - and the recipe has no check
// that could tell a restored database from a freshly migrated one anyway. See ADR-064.
const NoCountingCheck = "the empty stack never started, so no check was tested against " +
	"an empty application, and this recipe has no check that counts anything. Add one " +
	"check whose expect is scalar_int_min, rows_min, json_path_len_min, " +
	"json_path_int_min or glob_min_count, against data only a real installation has"

// Stage statuses.
const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusError   = "error"
	StatusSkipped = "skipped"
)

// Options is one `restored recipe test` invocation, for one recipe.
type Options struct {
	Recipe *recipe.Recipe

	// Stage is "a", "b" or "both".
	Stage string
	// Timeout is the whole harness budget for this recipe. The per-phase budgets of
	// SPEC.md section 7.4 are shares of it, so lowering it shortens every phase
	// rather than starving the last one.
	Timeout time.Duration

	Pull            string
	WorkspaceParent string
	Keep            bool
	ResticImage     string

	Version string
	Commit  string
	Debug   io.Writer
}

// Kept is a workspace and compose project the harness deliberately left behind.
type Kept struct {
	Stage     string `json:"stage"`
	Workspace string `json:"workspace"`
	Project   string `json:"compose_project"`
}

// Phase is one timed step inside a stage.
type Phase struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Note       string `json:"note,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Stage is one half of the round trip.
type Stage struct {
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	DurationMS int64   `json:"duration_ms"`
	Phases     []Phase `json:"phases,omitempty"`
	Error      string  `json:"error,omitempty"`
	// Command is what a human types to reproduce the stage's final step. It has to
	// still work after the stage has finished: the harness deletes its workspaces
	// unless --keep was passed, so a command naming one of them is a command that
	// answers "no such file or directory". See ADR-061.
	Command string `json:"command,omitempty"`
	// Check is the report from the `restored check` this stage ran, when there was
	// one. It carries the per-check query, expectation and observation, the service
	// logs and the hint - which is to say, everything a contributor needs to know
	// what went wrong, and all of which used to be computed and thrown away. It is
	// omitted for a stage that passed, because nobody reads a passing check's
	// observations and the JSON is uploaded as a CI artifact.
	// See docs/review/maintainer.md MNT-03 and docs/review/fresh-clone.md FC-05.
	Check *report.Report `json:"check,omitempty"`
}

// Result is one recipe, tested.
type Result struct {
	Recipe     string  `json:"recipe"`
	Title      string  `json:"title,omitempty"`
	Dir        string  `json:"dir,omitempty"`
	Status     string  `json:"status"`
	Stages     []Stage `json:"stages"`
	DurationMS int64   `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
	Kept       []Kept  `json:"kept,omitempty"`
}

// budget is the per-phase split of SPEC.md section 7.4, expressed as shares of the
// total so that --timeout scales the whole harness rather than one end of it.
type budget struct {
	stageA time.Duration
	seed   time.Duration
	export time.Duration
	check  time.Duration
}

func budgets(total time.Duration) budget {
	if total <= 0 {
		total = DefaultTimeout
	}
	share := func(n int) time.Duration { return total * time.Duration(n) / 20 }
	return budget{stageA: share(5), seed: share(5), export: share(3), check: share(7)}
}

func (o *Options) applyDefaults() {
	if o.Stage == "" {
		o.Stage = "both"
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.Pull == "" {
		o.Pull = "missing"
	}
	if o.ResticImage == "" {
		o.ResticImage = resticImage()
	}
	if o.Version == "" {
		o.Version = "0.0.0-dev"
	}
}

func resticImage() string {
	if v := os.Getenv("RESTORED_RESTIC_IMAGE"); v != "" {
		return v
	}
	return DefaultResticImage
}

// Run tests one recipe. A non-nil error is a tool error and maps to exit 2; so does a
// stage A that found no data-sensitive check, because a recipe that proves nothing is
// an invalid recipe rather than a failing one. A stage B failure maps to exit 1.
// See DECISIONS.md ADR-052.
func Run(ctx context.Context, o Options) (*Result, error) {
	o.applyDefaults()
	started := time.Now()

	res := &Result{
		Recipe: o.Recipe.Metadata.Name,
		Title:  o.Recipe.Metadata.Title,
		Dir:    o.Recipe.Dir,
		Status: StatusPass,
	}
	finish := func(err error) (*Result, error) {
		res.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			res.Status = StatusError
			res.Error = err.Error()
		}
		return res, err
	}

	switch o.Stage {
	case "a", "b", "both":
	default:
		return finish(fmt.Errorf("--stage %q: expected a, b or both", o.Stage))
	}
	// Stage B builds a restic repository of its own, so the harness needs restic for
	// any stage but A. It asks the restic source rather than asking docker.
	if o.Stage != "a" {
		if err := (&resticsource.Source{}).Preflight(ctx); err != nil {
			return finish(err)
		}
	}
	if err := compose.Preflight(ctx); err != nil {
		return finish(err)
	}
	if o.Recipe.Test == nil && o.Stage != "a" {
		return finish(fmt.Errorf("recipe %q has no test: section, so there is nothing to seed "+
			"or export; see SPEC.md section 7 and recipes/TEMPLATE/recipe.yaml",
			o.Recipe.Metadata.Name))
	}

	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	b := budgets(o.Timeout)

	if o.Stage == "a" || o.Stage == "both" {
		st, kept, err := o.stageA(ctx, b.stageA)
		res.Stages = append(res.Stages, st)
		res.Kept = appendKept(res.Kept, kept)
		if err != nil {
			return finish(o.timeoutError(ctx, err))
		}
		if st.Status == StatusFail {
			return finish(errors.New(st.Reason))
		}
	}
	if o.Stage == "b" || o.Stage == "both" {
		st, kept, err := o.stageB(ctx, b)
		res.Stages = append(res.Stages, st)
		res.Kept = appendKept(res.Kept, kept)
		if err != nil {
			return finish(o.timeoutError(ctx, err))
		}
		if st.Status != StatusPass {
			res.Status = StatusFail
			res.Error = st.Reason
		}
	}
	return finish(nil)
}

// timeoutError turns a cancelled context into the message a contributor can act on.
// A recipe that cannot round-trip inside its budget is out of scope for v0.1, and
// that is better said than discovered. See SPEC.md section 7.4.
func (o Options) timeoutError(ctx context.Context, err error) error {
	if ctx.Err() == nil {
		return err
	}
	return fmt.Errorf("recipe %q ran out of its %s budget: %w (raise it with --timeout, "+
		"but a recipe that needs longer is out of scope for v0.1 and its README should say so)",
		o.Recipe.Metadata.Name, o.Timeout, err)
}

func appendKept(all []Kept, k *Kept) []Kept {
	if k == nil {
		return all
	}
	return append(all, *k)
}

// stageA starts the stack with empty inputs and requires that at least one check
// fails. It runs the ordinary `check` code path against a tree of empty inputs, so
// there is no stage-A-only restore path that could pass while the real one is broken.
func (o Options) stageA(ctx context.Context, budget time.Duration) (st Stage, kept *Kept, err error) {
	st = Stage{Name: "A", Title: "negative: the checks must fail against an empty stack"}
	start := time.Now()
	defer func() { st.DurationMS = time.Since(start).Milliseconds() }()

	res, err := recipe.Resolve(o.Recipe, recipe.Options{
		InputsDir: filepath.Join("<workspace>", "inputs"),
		RunID:     "stage-a",
	})
	if err != nil {
		st.Status = StatusError
		st.Error = err.Error()
		return st, nil, err
	}

	parent := o.WorkspaceParent
	if parent == "" {
		parent = os.TempDir()
	}
	tree, mkErr := os.MkdirTemp(parent, "restored-empty-*")
	if mkErr != nil {
		st.Status = StatusError
		st.Error = mkErr.Error()
		return st, nil, fmt.Errorf("creating the empty input tree: %w", mkErr)
	}
	defer func() {
		if !o.Keep {
			_ = os.RemoveAll(tree)
		}
	}()
	if err := emptyTree(tree, res); err != nil {
		st.Status = StatusError
		st.Error = err.Error()
		return st, nil, err
	}

	// `tree` is deleted by the deferred cleanup above unless --keep was passed, so a
	// command naming it is a command that answers "no such file or directory". This
	// one rebuilds the stage. See ADR-061.
	st.Command = fmt.Sprintf("restored recipe test %s --stage a --keep", recipeRef(o.Recipe))

	phaseStart := time.Now()
	rep, innerKept, runErr := runner.Run(ctx, runner.Options{
		Recipe:          o.Recipe,
		SourceKind:      "dir",
		From:            tree,
		Timeout:         budget,
		ReadyTimeout:    stageAReady,
		Pull:            o.Pull,
		WorkspaceParent: o.WorkspaceParent,
		Keep:            o.Keep,
		Version:         o.Version,
		Commit:          o.Commit,
		Debug:           o.Debug,
	})
	phase := Phase{Name: "check against empty inputs", DurationMS: time.Since(phaseStart).Milliseconds()}

	if innerKept != nil {
		kept = &Kept{Stage: "A", Workspace: innerKept.Workspace, Project: innerKept.Project}
	}
	if runErr != nil {
		st.Check = rep
		phase.Status = StatusError
		phase.Error = runErr.Error()
		st.Phases = append(st.Phases, phase)
		st.Status = StatusError
		st.Error = runErr.Error()
		return st, kept, runErr
	}

	switch {
	case len(rep.Checks) == 0:
		// The application never became ready, so no check ran. Some applications
		// refuse to start with no data at all, which is itself evidence that the
		// checks are data-sensitive. SPEC.md section 7.2 calls this outcome
		// PASS-BY-STARTUP-REFUSAL, and it is reported as what it is.
		phase.Note = "the stack never became ready, so no check ran"
		// The stage passes, but nothing was proved by running: zero checks were
		// exercised negatively. ADR-032 was honest that this is weaker evidence and
		// "should look weaker" - and then nothing took over from the evidence that
		// was missing. A recipe whose checks are all `status: 200` and
		// `exists: true` could pass stage A by refusal, pass stage B on real data,
		// go green in CI, and pass against a schema-only dump: the exact false PASS
		// this tool exists to destroy, with a green badge on it.
		//
		// So when the run proves nothing, the recipe has to prove something about
		// itself. The expect vocabulary is closed precisely so a test like this can
		// be mechanical. See DECISIONS.md ADR-064.
		if ids := countingChecks(o.Recipe); len(ids) == 0 {
			phase.Status = StatusFail
			st.Status = StatusFail
			st.Reason = NoCountingCheck
			break
		}
		phase.Status = StatusPass
		st.Status = StatusPass
		st.Reason = "PASS-BY-STARTUP-REFUSAL: " + startupRefusalReason(rep)
	case rep.Summary.ChecksFailed > 0:
		phase.Status = StatusPass
		phase.Note = fmt.Sprintf("%d of %d checks failed", rep.Summary.ChecksFailed, rep.Summary.ChecksTotal)
		st.Status = StatusPass
		st.Reason = fmt.Sprintf("%d of %d checks failed against an empty stack: %s",
			rep.Summary.ChecksFailed, rep.Summary.ChecksTotal, failedIDs(rep))
	default:
		// Every check passed against an empty stack, so none of them is looking at
		// the data. The contributor's next question is "which of my checks is not
		// data-sensitive, and what did it see?", and the answer is in the report.
		st.Check = rep
		phase.Status = StatusFail
		phase.Note = fmt.Sprintf("all %d checks passed", rep.Summary.ChecksTotal)
		st.Status = StatusFail
		st.Reason = NoDataSensitiveCheck
	}
	st.Phases = append(st.Phases, phase)
	return st, kept, nil
}

// startupRefusalReason names the stage that stopped the empty stack, so a recipe
// author can tell "the app refuses to boot without data" apart from "my ready probe
// has the wrong URL in it".
func startupRefusalReason(rep *report.Report) string {
	for i := len(rep.Stages) - 1; i >= 0; i-- {
		if rep.Stages[i].Status == "ok" {
			continue
		}
		if rep.Stages[i].Error != "" {
			return rep.Stages[i].Name + ": " + rep.Stages[i].Error
		}
		return rep.Stages[i].Name + " did not complete"
	}
	return "no check ran"
}

func failedIDs(rep *report.Report) string {
	var ids []string
	for _, c := range rep.Checks {
		if c.Status != "pass" {
			ids = append(ids, c.ID)
		}
	}
	return strings.Join(ids, ", ")
}

// recipeRef is how a human would name this recipe on a command line.
func recipeRef(r *recipe.Recipe) string {
	if r.Bundled {
		return r.Metadata.Name
	}
	return r.Dir
}

// countingChecks returns the ids of the checks that could tell a restored database
// from an empty one that the application has just migrated for itself.
//
// The test is on the expect key rather than on the check kind, because that is where
// the distinction actually lives: `status: 200` and `exists: true` pass against an
// empty application, and `scalar_int_min: 1` does not. The vocabulary is closed
// (README.md, "a smaller expect vocabulary that a reviewer can hold in their head"),
// which is what makes this a list rather than a judgement.
func countingChecks(r *recipe.Recipe) []string {
	var ids []string
	for _, c := range r.Checks {
		e := c.Expect
		switch {
		case e.ScalarIntMin != nil,
			e.RowsMin != nil,
			e.JSONPathLenMin != nil,
			e.JSONPathIntMin != nil,
			e.GlobMinCount != nil,
			e.ScalarEquals != nil,
			e.JSONPathEquals != nil,
			e.SizeMin != nil,
			e.NotEmpty != nil,
			e.BodyMatches != "",
			e.StdoutMatches != "":
			ids = append(ids, c.ID)
		}
	}
	return ids
}
