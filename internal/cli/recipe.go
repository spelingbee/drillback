package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/recipe/safety"
)

func newRecipe(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recipe",
		Short: "Work with recipes: validate, show, init, test",
		Long: "Work with recipes.\n\n" +
			"A recipe is a directory containing recipe.yaml, compose.yaml, and optionally\n" +
			"test assets. It declares the logical inputs an application needs — not your paths —\n" +
			"plus the probes that say the app is up and the checks that say the data survived.",
	}
	cmd.AddCommand(newRecipeValidate(g), newRecipeShow(g), newRecipeInit(g), newRecipeTest(g))
	AddExitCodes(cmd, RecipeExitCodes)
	return cmd
}

// ---- validate --------------------------------------------------------------

type finding struct {
	Recipe   string   `json:"recipe"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func newRecipeValidate(g *globals) *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate <dir|file>...",
		Short: "Validate a recipe against the schema and the safety rules",
		Long: "Validate one or more recipes against the JSON Schema and the safety rules.\n\n" +
			"The safety rules are hard failures, never warnings. A recipe is rejected if its\n" +
			"compose.yaml uses privileged containers, host networking, the host PID or IPC\n" +
			"namespace, published ports, a non-internal network, a bind mount to a path outside\n" +
			"the run workspace, or a YAML tag. See SPEC.md section 9, Threat model.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			findings := make([]finding, 0, len(args))
			bad := false
			for _, ref := range args {
				f := validateOne(ref, strict)
				if !f.Valid || (strict && len(f.Warnings) > 0) {
					bad = true
				}
				findings = append(findings, f)
			}
			if g.json {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(findings); err != nil {
					return fail(ExitError, "%v", err)
				}
			} else {
				for _, f := range findings {
					printFinding(cmd, f, strict)
				}
			}
			if bad {
				// Say why the exit code is what it is. Under --strict, "invalid" and
				// "valid but warned" both exit 2, and the difference is invisible
				// otherwise - so a `ok` line above a exit 2 read as a bug in the tool
				// rather than as the answer. See docs/review/ux.md UX-07.
				if !g.json {
					invalid, warned := 0, 0
					for _, f := range findings {
						switch {
						case !f.Valid:
							invalid++
						case len(f.Warnings) > 0:
							warned++
						}
					}
					out := cmd.OutOrStdout()
					switch {
					case invalid > 0 && warned > 0:
						fmt.Fprintf(out, "\n%d invalid, %d with warnings. Exit 2.\n", invalid, warned)
					case invalid > 0:
						fmt.Fprintf(out, "\n%d invalid. Exit 2.\n", invalid)
					default:
						fmt.Fprintf(out, "\n%d recipe(s) with warnings, and --strict makes a "+
							"warning a failure. Exit 2.\nWithout --strict these are advisory "+
							"and this command exits 0.\n", warned)
					}
				}
				return &exitError{code: ExitError, err: errSilent{}}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false,
		"Also fail on warnings: missing maintainer, an image reference without a tag, a recipe with fewer than two checks")
	AddExitCodes(cmd, RecipeExitCodes)
	return cmd
}

func validateOne(ref string, strict bool) finding {
	f := finding{Recipe: ref}
	rec, err := recipe.LoadAny(ref)
	if err != nil {
		f.Errors = append(f.Errors, err.Error())
		return f
	}
	composeRaw, err := rec.ReadFile("compose.yaml")
	if err != nil {
		f.Errors = append(f.Errors, err.Error())
		return f
	}
	res, err := recipe.Resolve(rec, recipe.Options{
		InputsDir: filepath.Join("<workspace>", "inputs"),
		RunID:     "validate",
	})
	if err != nil {
		f.Errors = append(f.Errors, err.Error())
		return f
	}
	if err := safety.Validate(composeRaw, res); err != nil {
		f.Errors = append(f.Errors, err.Error())
		return f
	}
	f.Valid = true
	f.Warnings = safety.Warnings(rec, composeRaw)
	_ = strict
	return f
}

// printFinding renders one recipe's verdict. Under --strict a warning is a failure, so
// the status word has to say so, and every warning is indented under the recipe it
// belongs to: `recipe validate ./recipes/*/` prints six of these, and an unattributed
// `warning` line in the middle of that list belongs to nobody.
// See docs/review/ux.md UX-07.
func printFinding(cmd *cobra.Command, f finding, strict bool) {
	w := cmd.OutOrStdout()
	switch {
	case !f.Valid:
		_, _ = fmt.Fprintf(w, "INVALID  %s\n", f.Recipe)
		for _, e := range f.Errors {
			_, _ = fmt.Fprintf(w, "         %s\n", e)
		}
	case strict && len(f.Warnings) > 0:
		_, _ = fmt.Fprintf(w, "WARN     %s\n", f.Recipe)
	default:
		fmt.Fprintf(w, "ok       %s\n", f.Recipe)
	}
	for _, warn := range f.Warnings {
		fmt.Fprintf(w, "         warning: %s\n", warn)
	}
}

// ---- show ------------------------------------------------------------------

func newRecipeShow(g *globals) *cobra.Command {
	var (
		format      string
		inputs      []string
		sets        []string
		showCompose bool
		inputsOnly  bool
	)
	cmd := &cobra.Command{
		Use:   "show <name|dir|file>",
		Short: "Print a resolved recipe: defaults applied, variables expanded",
		Long: "Print a recipe with defaults applied, variables expanded, and inputs resolved to the\n" +
			"paths this invocation would actually use.\n\n" +
			"Use this to see exactly what `restored check` would do, without running anything.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --json is a global flag, advertised under Global Flags in this
			// command's own help, and it used to emit YAML. Two flag names mean JSON
			// in this tool; on this command they now mean the same thing. See
			// docs/review/ux.md UX-05.
			if g.json {
				format = "json"
			}
			inputMap, err := keyValues(inputs, "input")
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			setMap, err := keyValues(sets, "set")
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			rec, err := recipe.LoadAny(args[0])
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			res, err := recipe.Resolve(rec, recipe.Options{
				InputPaths: inputMap,
				Vars:       setMap,
				InputsDir:  filepath.Join("<workspace>", "inputs"),
				RunID:      "<runid>",
			})
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			// Read once here rather than inside showDocument: the images have to be
			// disclosed on every path, and --compose below needs the same bytes.
			composeRaw, err := rec.ReadFile("compose.yaml")
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			w := cmd.OutOrStdout()

			if inputsOnly {
				return writeInputTable(w, res, format)
			}
			switch format {
			case "json":
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				if err := enc.Encode(showDocument(res, composeRaw)); err != nil {
					return fail(ExitError, "%v", err)
				}
			case "yaml", "":
				out, err := yaml.Marshal(showDocument(res, composeRaw))
				if err != nil {
					return fail(ExitError, "%v", err)
				}
				if _, err := w.Write(out); err != nil {
					return fail(ExitError, "%v", err)
				}
			default:
				return fail(ExitError, "--format %q: expected yaml or json", format)
			}
			if showCompose {
				rendered, err := safety.Render(composeRaw, res.ComposeEnv())
				if err != nil {
					return fail(ExitError, "%v", err)
				}
				fmt.Fprintf(w, "\n# --- rendered compose.yaml ---\n%s", rendered)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "yaml",
		"yaml|json. --json is equivalent to --format json")
	cmd.Flags().StringArrayVar(&inputs, "input", nil, "Override an input path (repeatable), as name=path")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "Override a variable (repeatable), as key=value")
	cmd.Flags().BoolVar(&showCompose, "compose", false, "Also print the rendered compose.yaml")
	cmd.Flags().BoolVar(&inputsOnly, "inputs-only", false,
		"Print only the resolved input table — the fastest way to answer \"which paths does this recipe want from my backup?\"")
	AddExitCodes(cmd, RecipeExitCodes)
	return cmd
}

type shownInput struct {
	Name       string `json:"name" yaml:"name"`
	Kind       string `json:"kind" yaml:"kind"`
	Title      string `json:"title" yaml:"title"`
	BackupPath string `json:"backup_path" yaml:"backup_path"`
	LocalPath  string `json:"local_path" yaml:"local_path"`
	Origin     string `json:"source" yaml:"source"`
	Required   bool   `json:"required" yaml:"required"`
}

type shownRecipe struct {
	Metadata recipe.Metadata `json:"metadata" yaml:"metadata"`
	// Images is every container image this recipe will pull, listed before anything
	// else a reader might act on. Running a recipe runs somebody else's images, which
	// SPEC.md section 9.2 records as an accepted risk on the grounds that `recipe
	// show` discloses them - and it did not disclose them at all until the session 4
	// security review pointed out that the control the argument rests on was missing
	// (docs/review/security.md SEC-07).
	Images []string        `json:"images" yaml:"images"`
	Vars   map[string]any  `json:"vars,omitempty" yaml:"vars,omitempty"`
	Inputs []shownInput    `json:"inputs" yaml:"inputs"`
	Ready  []*recipe.Probe `json:"ready,omitempty" yaml:"ready,omitempty"`
	Checks []*recipe.Check `json:"checks" yaml:"checks"`
}

func showDocument(res *recipe.Resolved, composeRaw []byte) shownRecipe {
	return shownRecipe{
		Metadata: res.Recipe.Metadata,
		Images:   recipeImages(composeRaw),
		Vars:     res.Vars,
		Inputs:   shownInputs(res),
		Ready:    res.Recipe.Ready,
		Checks:   res.Recipe.Checks,
	}
}

func shownInputs(res *recipe.Resolved) []shownInput {
	out := make([]shownInput, 0, len(res.Inputs))
	for _, in := range res.Inputs {
		out = append(out, shownInput{
			Name: in.Name, Kind: in.Kind, Title: in.Title,
			BackupPath: in.BackupPath, LocalPath: in.LocalPath,
			Origin: string(in.Origin), Required: in.Required,
		})
	}
	return out
}

func writeInputTable(w interface{ Write([]byte) (int, error) }, res *recipe.Resolved, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(shownInputs(res)); err != nil {
			return fail(ExitError, "%v", err)
		}
		return nil
	}
	nameW, kindW := 0, 0
	for _, in := range res.Inputs {
		if len(in.Name) > nameW {
			nameW = len(in.Name)
		}
		if len(in.Kind) > kindW {
			kindW = len(in.Kind)
		}
	}
	for _, in := range res.Inputs {
		req := "optional"
		if in.Required {
			req = "required"
		}
		fmt.Fprintf(w, "%-*s  %-*s  %-8s  %s\n", nameW, in.Name, kindW, in.Kind, req, in.BackupPath)
	}
	return nil
}

// ---- init ------------------------------------------------------------------

func newRecipeInit(g *globals) *cobra.Command {
	var (
		dir      string
		db       string
		withDirs []string
		image    string
		force    bool
		from     string
	)
	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Scaffold a new recipe directory",
		Long: "Scaffold a new recipe directory: recipe.yaml, compose.yaml, and README.md, prefilled\n" +
			"with comments and a working skeleton.\n\n" +
			"The generated recipe deliberately does NOT prove anything yet. It has no\n" +
			"data-sensitive check, and writing that check is the first and most important thing\n" +
			"you do, because it is the only thing that makes the recipe worth anything.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Rejected rather than ignored. `recipe init` writes files and prints
			// what it wrote; there is no machine-readable version of that, and
			// accepting the flag silently teaches a script that it worked.
			if g.json {
				return fail(ExitError,
					"--json: recipe init has no machine-readable output. It writes files "+
						"and prints what it wrote; `restored recipe show <dir> --json` will "+
						"read the result back")
			}
			name := args[0]
			if from != "" {
				return initFromCompose(cmd, name, dir, from, force)
			}
			switch db {
			case "none", "sqlite", "postgres-dump":
			default:
				return fail(ExitError, "--db %q: expected none, sqlite or postgres-dump", db)
			}
			target := filepath.Join(dir, name)
			if _, err := os.Stat(target); err == nil && !force {
				return fail(ExitError, "%s already exists; pass --force to overwrite", target)
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fail(ExitError, "%v", err)
			}
			files, err := scaffold(name, db, image, withDirs)
			if err != nil {
				return fail(ExitError, "%v", err)
			}
			for file, body := range files {
				if err := os.WriteFile(filepath.Join(target, file), []byte(body), 0o644); err != nil {
					return fail(ExitError, "%v", err)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n\nNext:\n"+
				"  1. make the checks data-sensitive: a check that passes against an empty\n"+
				"     database proves nothing about a restore\n"+
				"  2. restored recipe validate %s --strict\n", target, target)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "./recipes", "Parent directory")
	cmd.Flags().StringVar(&db, "db", "none", "Database input to scaffold: none|sqlite|postgres-dump")
	cmd.Flags().StringSliceVar(&withDirs, "with-dir", []string{"data"}, "Scaffold a dir input with this name (repeatable)")
	cmd.Flags().StringVar(&image, "image", "", "Application image to write into compose.yaml")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing directory")
	cmd.Flags().StringVar(&from, "compose", "",
		"Read an existing compose file and propose a recipe from it: volumes become dir inputs, "+
			"a database service becomes a dump or sqlite input, an exposed port becomes the ready probe")
	AddExitCodes(cmd, RecipeExitCodes)
	return cmd
}

// recipeImages lists every image the recipe's compose file names, sorted and
// deduplicated. It reads the compose file rather than the safety schema's typed view
// so that a service the schema would reject is still disclosed: the point is to show
// what would be pulled, not what would be allowed.
func recipeImages(composeRaw []byte) []string {
	var doc struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(composeRaw, &doc); err != nil {
		return []string{"(compose.yaml could not be parsed, so the images are unknown)"}
	}
	seen := map[string]bool{}
	var out []string
	for _, svc := range doc.Services {
		if svc.Image == "" || seen[svc.Image] {
			continue
		}
		seen[svc.Image] = true
		out = append(out, svc.Image)
	}
	sort.Strings(out)
	return out
}
