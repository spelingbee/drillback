package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// initFromCompose is `recipe init --compose`: read somebody's real compose file and
// write the recipe it implies, with a TODO wherever the answer is theirs to give.
//
// The summary it prints matters as much as the files. A contributor should be able to
// see, without opening anything, what this command believed about their application -
// and therefore where it is most likely to be wrong.
func initFromCompose(cmd *cobra.Command, name, dir, from string, force bool) error {
	d, err := detectCompose(from)
	if err != nil {
		return fail(ExitError, "%v", err)
	}
	files, err := scaffoldFromCompose(name, d)
	if err != nil {
		return fail(ExitError, "%v", err)
	}
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); err == nil && !force {
		return fail(ExitError, "%s already exists; pass --force to overwrite", target)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fail(ExitError, "%v", err)
	}
	for file, body := range files {
		if err := os.WriteFile(filepath.Join(target, file), []byte(body), 0o644); err != nil {
			return fail(ExitError, "%v", err)
		}
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Wrote %s from %s\n\n", target, from)
	fmt.Fprintf(w, "  application:  %s (%s), port %d\n", d.AppService, d.AppImage, d.AppPort)
	for _, dd := range d.Dirs {
		fmt.Fprintf(w, "  dir input:    %s from %s:%s\n", dd.Name, dd.Service, dd.Container)
	}
	if d.DB != nil {
		fmt.Fprintf(w, "  database:     %s in service %s\n", d.DB.Kind, d.DB.Service)
	}
	for _, note := range d.Notes {
		for i, line := range wrapComment(note, 60) {
			label := "  note:         "
			if i > 0 {
				label = "                "
			}
			fmt.Fprintf(w, "%s%s\n", label, line)
		}
	}
	fmt.Fprintf(w, `
This is a proposal, not a recipe. Next:

  1. work through the TODO markers in %s
  2. make one check data-sensitive: a check that passes against an empty database
     proves nothing about a restore, and the round-trip harness will refuse it
  3. restored recipe validate %s
  4. restored recipe test %s     # this is what CI runs
`, filepath.Join(target, "recipe.yaml"), target, target)
	return nil
}
