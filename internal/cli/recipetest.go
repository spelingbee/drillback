package cli

import (
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/spelingbee/restored/internal/harness"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/report"
)

func newRecipeTest(g *globals) *cobra.Command {
	var (
		stage      string
		timeout    time.Duration
		keep       bool
		reportFile string
		pull       string
		workspace  string
	)
	cmd := &cobra.Command{
		Use:   "test <name|dir>...",
		Short: "Run the round-trip harness against a recipe",
		Long: "Run the round-trip harness against a recipe. This is what CI runs for every PR\n" +
			"that touches recipes/**, and it runs identically on your laptop.\n\n" +
			"Stage A, \"negative\": start the stack with EMPTY inputs, run the checks, and\n" +
			"require that AT LEAST ONE check FAILS. A recipe whose checks all pass against an\n" +
			"empty stack proves nothing about a restore and is rejected with \"recipe has no\n" +
			"data-sensitive check\".\n\n" +
			"Stage B, \"positive\": start a fresh stack, run test.seed, run test.export, back\n" +
			"the resulting input tree up into a throwaway restic repository, tear everything\n" +
			"down, then run a normal `restored check` against that repository and require that\n" +
			"ALL checks PASS.",
		Example: "  restored recipe test ./recipes/gitea\n" +
			"  restored recipe test ./recipes/gitea --stage a --keep\n" +
			"  restored recipe test ./recipes/* --json --report ./recipe-test.json",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch stage {
			case "a", "b", "both":
			default:
				return fail(ExitError, "--stage %q: expected a, b or both", stage)
			}
			switch pull {
			case "always", "missing", "never":
			default:
				return fail(ExitError, "--pull %q: expected always, missing or never", pull)
			}

			started := time.Now()
			rep := &harness.Report{
				SchemaVersion: harness.SchemaVersion,
				Tool:          harness.Tool{Name: "restored", Version: Version, Commit: Commit},
				StartedAt:     started.UTC().Format(time.RFC3339),
			}
			worst := ExitPass
			for _, ref := range args {
				res, err := testOne(cmd, ref, harness.Options{
					Stage:           stage,
					Timeout:         timeout,
					Keep:            keep,
					Pull:            pull,
					WorkspaceParent: workspace,
					Version:         Version,
					Commit:          Commit,
					Debug:           debugWriter(g),
				})
				rep.Add(res)
				switch {
				case err != nil:
					worst = ExitError
				case res.Status != harness.StatusPass && worst == ExitPass:
					worst = ExitUnusable
				}
			}
			rep.FinishedAt = time.Now().UTC().Format(time.RFC3339)
			rep.DurationMS = time.Since(started).Milliseconds()

			human := cmd.OutOrStdout()
			if g.json {
				human = cmd.ErrOrStderr()
			}
			if err := rep.WriteTTY(human, report.Options{
				Color: colourEnabled(g, human),
				ASCII: os.Getenv("RESTORED_ASCII") != "",
			}); err != nil {
				return fail(ExitError, "%v", err)
			}
			if g.json {
				if err := rep.WriteJSON(cmd.OutOrStdout()); err != nil {
					return fail(ExitError, "%v", err)
				}
			}
			if reportFile != "" {
				if err := writeJSONFile(reportFile, rep); err != nil {
					return fail(ExitError, "writing --report: %v", err)
				}
			}
			if worst != ExitPass {
				return &exitError{code: worst, err: errSilent{}}
			}
			return nil
		},
	}
	fl := cmd.Flags()
	fl.StringVar(&stage, "stage", "both", "a|b|both")
	fl.DurationVar(&timeout, "timeout", harness.DefaultTimeout,
		"Wall-clock budget per recipe for the whole harness")
	fl.BoolVar(&keep, "keep", false, "Keep workspaces and compose projects for inspection")
	fl.StringVar(&reportFile, "report", "", "Write the JSON report to this file")
	fl.StringVar(&pull, "pull", "missing", "Image pull policy: always|missing|never")
	fl.StringVar(&workspace, "workspace", "",
		"Parent directory for the run workspaces (default: the OS temp directory)")
	cmd.SetHelpTemplate(cmd.HelpTemplate() + `
Exit codes:
  0  every recipe passed both stages
  1  stage B failed: the round trip did not restore
  2  tool error, or stage A found no data-sensitive check, which makes the recipe
     invalid rather than failing

Docs: https://github.com/spelingbee/restored/blob/main/CONTRIBUTING.md
`)
	return cmd
}

// testOne loads and tests one recipe. A recipe that cannot be loaded is reported as an
// errored result rather than aborting the whole invocation, so `recipe test
// ./recipes/*` gives a verdict for every recipe named rather than for the ones before
// the first broken one.
func testOne(cmd *cobra.Command, ref string, o harness.Options) (*harness.Result, error) {
	rec, err := recipe.LoadAny(ref)
	if err != nil {
		return &harness.Result{Recipe: ref, Status: harness.StatusError, Error: err.Error()}, err
	}
	o.Recipe = rec
	return harness.Run(cmd.Context(), o)
}

func writeJSONFile(path string, rep *harness.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := rep.WriteJSON(f)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// debugWriter sends the harness's command lines to stderr when the user asked for
// them. They reach the run's own log file either way.
func debugWriter(g *globals) io.Writer {
	if g.logLevel == "debug" || g.logLevel == "trace" {
		return os.Stderr
	}
	return nil
}
