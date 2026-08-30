package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TargetRun is one element of the --all document: exactly the single-run report,
// plus the name of the target that produced it. The addition is what the stability
// contract allows - fields are added, never removed or retyped - and without it a
// consumer cannot tell two targets sharing a recipe apart. See ADR-068.
type TargetRun struct {
	Target string `json:"target"`
	*Report
}

// Multi is the --all document of SPEC.md section 5.2: every run, a summary across
// targets, and the worst exit code.
type Multi struct {
	SchemaVersion int          `json:"schema_version"`
	Runs          []TargetRun  `json:"runs"`
	Summary       MultiSummary `json:"summary"`
	ExitCode      int          `json:"exit_code"`
}

// MultiSummary counts targets, not checks: the per-run summaries already count
// checks, and the question a cron consumer asks of the aggregate is "how many of my
// backups are proven, and how many runs told me nothing".
type MultiSummary struct {
	TargetsTotal    int   `json:"targets_total"`
	TargetsPassed   int   `json:"targets_passed"`
	TargetsUnusable int   `json:"targets_unusable"`
	TargetsErrored  int   `json:"targets_errored"`
	DurationMS      int64 `json:"duration_ms"`
}

// NewMulti returns an empty --all document.
func NewMulti() *Multi {
	return &Multi{SchemaVersion: SchemaVersion}
}

// Add appends one target's report and folds it into the summary and the exit code.
// The worst outcome wins: 2 if any target hit a tool error, else 1 if any restore
// was unusable, else 0 (SPEC.md 2.9).
func (m *Multi) Add(target string, r *Report) {
	m.Runs = append(m.Runs, TargetRun{Target: target, Report: r})
	m.Summary.TargetsTotal++
	m.Summary.DurationMS += r.Run.DurationMS
	switch r.Verdict {
	case VerdictPass:
		m.Summary.TargetsPassed++
	case VerdictUnusable:
		m.Summary.TargetsUnusable++
		if m.ExitCode < 1 {
			m.ExitCode = 1
		}
	default:
		m.Summary.TargetsErrored++
		m.ExitCode = 2
	}
}

// WriteJSON writes the whole document.
func (m *Multi) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("writing the JSON report: %w", err)
	}
	return nil
}

// WriteTTY renders the closing summary block. The per-run reports have already been
// rendered above it, one by one, as each target finished; this is the part a person
// scrolls to the bottom for.
func (m *Multi) WriteTTY(w io.Writer, o Options) error {
	g := o.glyphs()
	b := &strings.Builder{}

	width := 0
	for _, run := range m.Runs {
		if len(run.Target) > width {
			width = len(run.Target)
		}
	}

	fmt.Fprintln(b)
	for _, run := range m.Runs {
		verdict, colour := "PASS", colGreen
		note := ""
		switch run.Verdict {
		case VerdictPass:
		case VerdictUnusable:
			verdict, colour = "RESTORE UNUSABLE", colRed
			note = fmt.Sprintf("%d/%d checks", run.Summary.ChecksPassed, run.Summary.ChecksTotal)
		default:
			verdict, colour = "ERROR", colRed
			note = firstLine(run.Error)
		}
		// The verdict is padded before painting: ANSI codes have width on paper and
		// none on screen, so painting first would misalign every coloured line.
		fmt.Fprintf(b, "  target %-*s  %s  %6s", width, run.Target,
			o.paint(fmt.Sprintf("%-16s", verdict), colour+colBold), duration(run.Run.DurationMS))
		if note != "" {
			fmt.Fprintf(b, "  %s %s", g.dot, note)
		}
		fmt.Fprintln(b)
	}

	s := m.Summary
	fmt.Fprintf(b, "\n  %d targets: %d passed, %d unusable, %d errored, in %s\n",
		s.TargetsTotal, s.TargetsPassed, s.TargetsUnusable, s.TargetsErrored, duration(s.DurationMS))

	_, err := io.WriteString(w, b.String())
	return err
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const maxLen = 40
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}
