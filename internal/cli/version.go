package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/spelingbee/restored/internal/compose"
	"github.com/spelingbee/restored/internal/recipe"
)

func newVersion(g *globals) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v := compose.Probe(cmd.Context())
			bundled := recipe.BundledNames()
			w := cmd.OutOrStdout()

			if asJSON || g.json {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"version":  Version,
					"commit":   Commit,
					"built":    Date,
					"go":       runtime.Version(),
					"platform": runtime.GOOS + "/" + runtime.GOARCH,
					"docker":   orNotFound(v.Docker),
					"compose":  orNotFound(v.Compose),
					"restic":   orNotFound(v.Restic),
					"recipes":  bundled,
				})
			}

			fmt.Fprintf(w, "restored %s\n", Version)
			fmt.Fprintf(w, "  commit:    %s\n", orUnknown(Commit))
			fmt.Fprintf(w, "  built:     %s\n", orUnknown(Date))
			fmt.Fprintf(w, "  go:        %s\n", runtime.Version())
			fmt.Fprintf(w, "  platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(w, "  docker:    %s\n", dockerLine(v))
			fmt.Fprintf(w, "  restic:    %s\n", orNotFound(v.Restic))
			fmt.Fprintf(w, "  recipes:   %d bundled\n", len(bundled))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable output")
	return cmd
}

// dockerLine keeps docker and compose on one line, the way `restored version` is meant
// to be pasted into a bug report.
func dockerLine(v compose.Versions) string {
	if v.Docker == "" {
		return "not found"
	}
	if v.Compose == "" {
		return v.Docker + " (compose not found)"
	}
	return fmt.Sprintf("%s (compose v%s)", v.Docker, v.Compose)
}

func orNotFound(s string) string {
	if s == "" {
		return "not found"
	}
	return s
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
