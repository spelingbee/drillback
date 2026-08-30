// Package report is a pure function of its input struct. It does no I/O beyond
// writing to a supplied io.Writer, and it never reaches back into check or compose.
// That is what makes golden-file tests of the output possible at all.
package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spelingbee/drillback/internal/observe"
	"github.com/spelingbee/drillback/internal/recipe"
	"github.com/spelingbee/drillback/internal/source"
)

// SchemaVersion is bumped on any breaking change to the JSON document. Within a major
// version fields are only added, never removed or retyped.
const SchemaVersion = 1

// The three verdicts a run can end with.
const (
	VerdictPass     = "PASS"
	VerdictUnusable = "RESTORE_UNUSABLE"
	VerdictError    = "ERROR"
)

// Report is the whole record of one run. It is what --json prints and what --report
// writes, and the TTY output is rendered from nothing else.
type Report struct {
	SchemaVersion int                 `json:"schema_version"`
	Tool          Tool                `json:"tool"`
	Run           Run                 `json:"run"`
	Verdict       string              `json:"verdict"`
	ExitCode      int                 `json:"exit_code"`
	Recipe        RecipeInfo          `json:"recipe"`
	Source        source.Descriptor   `json:"source"`
	Inputs        []Input             `json:"inputs"`
	Stages        []Stage             `json:"stages"`
	Checks        []Check             `json:"checks"`
	Summary       Summary             `json:"summary"`
	Hint          *Hint               `json:"hint,omitempty"`
	Warnings      []Warning           `json:"warnings,omitempty"`
	Logs          map[string][]string `json:"logs,omitempty"`
	Error         string              `json:"error,omitempty"`
}

// Tool identifies the binary that produced the report.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

// Run is the identity and timing of one run.
type Run struct {
	ID               string `json:"id"`
	ComposeProject   string `json:"compose_project"`
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at"`
	DurationMS       int64  `json:"duration_ms"`
	WorkspaceRemoved bool   `json:"workspace_removed"`
	Workspace        string `json:"workspace,omitempty"`
}

// RecipeInfo says which recipe ran and where it came from.
type RecipeInfo struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Source string `json:"source"`
	Digest string `json:"digest"`
}

// Input is one materialised input as the report shows it.
type Input struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	BackupPath     string `json:"backup_path"`
	Bytes          int64  `json:"bytes"`
	Files          int    `json:"files"`
	DetectedFormat string `json:"detected_format,omitempty"`
	Origin         string `json:"source"`
}

// Stage is one state of the run lifecycle.
type Stage struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	DurationMS int64          `json:"duration_ms"`
	Services   []string       `json:"services,omitempty"`
	Detail     map[string]any `json:"detail,omitempty"`
	Probes     []ProbeResult  `json:"probes,omitempty"`
	Error      string         `json:"error,omitempty"`
	// Note is the short human summary the TTY line carries.
	Note string `json:"note,omitempty"`
}

// ProbeResult mirrors one ready probe.
type ProbeResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Check is one check as the report shows it.
type Check struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	Kind       string              `json:"kind"`
	Status     string              `json:"status"`
	DurationMS int64               `json:"duration_ms"`
	Expect     recipe.Expect       `json:"expect"`
	Observed   observe.Observation `json:"observed"`
	Query      string              `json:"query,omitempty"`
	URL        string              `json:"url,omitempty"`
	Failures   []observe.Failure   `json:"failures,omitempty"`
}

// Summary is the count automation depends on.
type Summary struct {
	ChecksTotal   int `json:"checks_total"`
	ChecksPassed  int `json:"checks_passed"`
	ChecksFailed  int `json:"checks_failed"`
	ChecksSkipped int `json:"checks_skipped"`
}

// Hint is the single likely cause the report may print. A hint is presentation only:
// it can never change the verdict or the exit code.
type Hint struct {
	ID        string   `json:"id"`
	MatchedOn string   `json:"matched_on"`
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	Commands  []string `json:"commands,omitempty"`
}

// Warning is something the run changed or refused on the way through.
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// WriteJSON writes the report as the stable document described in SPEC.md section 5.2.
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("writing the JSON report: %w", err)
	}
	return nil
}
