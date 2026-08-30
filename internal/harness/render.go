package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spelingbee/restored/internal/report"
)

// SchemaVersion is the harness report's stability contract, separate from the check
// report's but deliberately the same field name and the same type. It changes only
// when a field is removed or its meaning changes.
//
// It was a string while report.SchemaVersion was an int, so `jq -e '.schema_version
// == 1'` passed against one document from this binary and failed against the other,
// for a reason nothing in either output explained. See docs/review/ux.md UX-06.
const SchemaVersion = 1

// Report is what `restored recipe test --json` writes: one document for the whole
// invocation, however many recipes it covered.
type Report struct {
	SchemaVersion int      `json:"schema_version"`
	Tool          Tool     `json:"tool"`
	StartedAt     string   `json:"started_at"`
	FinishedAt    string   `json:"finished_at"`
	DurationMS    int64    `json:"duration_ms"`
	Summary       Summary  `json:"summary"`
	Recipes       []Result `json:"recipes"`
}

// Tool identifies the build that produced the report.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

// Summary is the count a CI job reads without parsing the rest.
type Summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Errored int `json:"errored"`
}

// Add records one recipe's result and keeps the summary in step.
func (r *Report) Add(res *Result) {
	r.Recipes = append(r.Recipes, *res)
	r.Summary.Total++
	switch res.Status {
	case StatusPass:
		r.Summary.Passed++
	case StatusFail:
		r.Summary.Failed++
	default:
		r.Summary.Errored++
	}
}

// WriteJSON writes the machine-readable report.
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteTTY renders the human report. The verdict reads identically through NO_COLOR
// and through `| cat`: colour is an enhancement, never the only signal.
func (r *Report) WriteTTY(w io.Writer, o report.Options) error {
	b := &strings.Builder{}
	for _, res := range r.Recipes {
		writeResult(b, o, res)
	}
	fmt.Fprintf(b, "  %s\n", strings.Repeat("─", 4))
	fmt.Fprintf(b, "  %d recipe%s: %d passed, %d failed, %d errored, in %s\n\n",
		r.Summary.Total, plural(r.Summary.Total),
		r.Summary.Passed, r.Summary.Failed, r.Summary.Errored, duration(r.DurationMS))
	_, err := io.WriteString(w, b.String())
	return err
}

func writeResult(b *strings.Builder, o report.Options, res Result) {
	name := res.Recipe
	if res.Title != "" {
		name = fmt.Sprintf("%s (%s)", res.Recipe, res.Title)
	}
	fmt.Fprintf(b, "\nrecipe test %s\n\n", name)

	for _, st := range res.Stages {
		fmt.Fprintf(b, "  %-8s %-52s %s  %8s\n",
			"stage "+st.Name, truncate(st.Title, 52),
			paintStatus(o, st.Status), duration(st.DurationMS))
		if st.Reason != "" {
			for _, line := range wrap(st.Reason, 66) {
				fmt.Fprintf(b, "           %s\n", line)
			}
		}
		if st.Error != "" {
			for _, line := range wrap(st.Error, 66) {
				fmt.Fprintf(b, "           %s\n", paint(o, line, colRed))
			}
		}
		if len(st.Phases) > 0 {
			fmt.Fprintf(b, "           %s\n", paint(o, phaseLine(st.Phases), colDim))
		}
		if st.Command != "" {
			fmt.Fprintf(b, "           %s\n", paint(o, "$ "+st.Command, colDim))
		}
	}
	if res.Error != "" && len(res.Stages) == 0 {
		for _, line := range wrap(res.Error, 74) {
			fmt.Fprintf(b, "  %s\n", paint(o, line, colRed))
		}
	}
	for _, k := range res.Kept {
		fmt.Fprintf(b, "\n  kept (stage %s):  %s\n  compose project:  %s\n", k.Stage, k.Workspace, k.Project)
	}
	fmt.Fprintf(b, "\n  %s  %s in %s\n\n",
		paintStatus(o, res.Status), res.Recipe, duration(res.DurationMS))
}

// phaseLine is the per-phase timing, on one line, in the order the phases ran.
func phaseLine(phases []Phase) string {
	parts := make([]string, 0, len(phases))
	for _, p := range phases {
		parts = append(parts, fmt.Sprintf("%s %s", p.Name, duration(p.DurationMS)))
	}
	return strings.Join(parts, " · ")
}

const (
	colReset = "\x1b[0m"
	colGreen = "\x1b[32m"
	colRed   = "\x1b[31m"
	colDim   = "\x1b[2m"
)

func paintStatus(o report.Options, status string) string {
	label := map[string]string{
		StatusPass: "PASS", StatusFail: "FAIL", StatusError: "ERROR", StatusSkipped: "SKIP",
	}[status]
	if label == "" {
		label = strings.ToUpper(status)
	}
	// The colour is applied after the padding, never before: escape bytes count
	// towards a width and would push every column along.
	padded := fmt.Sprintf("%-5s", label)
	if status == StatusPass {
		return paint(o, padded, colGreen)
	}
	return paint(o, padded, colRed)
}

func paint(o report.Options, s, colour string) string {
	if !o.Color {
		return s
	}
	return colour + s + colReset
}

func truncate(s string, w int) string {
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "…"
}

func wrap(s string, w int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > w {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	return append(lines, line)
}

func duration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}
