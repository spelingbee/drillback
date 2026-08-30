package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Options control presentation only. The verdict reads identically through NO_COLOR,
// through `| cat`, and in a screenshot pasted into an issue: colour is an
// enhancement, never the only signal.
type Options struct {
	Color bool
	ASCII bool
}

// Width is the column the report is laid out against.
const Width = 78

type glyphs struct {
	pass, fail, arrow, dot string
}

func (o Options) glyphs() glyphs {
	if o.ASCII {
		return glyphs{pass: "+", fail: "x", arrow: "->", dot: "|"}
	}
	return glyphs{pass: "✔", fail: "✘", arrow: "→", dot: "·"}
}

const (
	colReset = "\x1b[0m"
	colGreen = "\x1b[32m"
	colRed   = "\x1b[31m"
	colDim   = "\x1b[2m"
	colBold  = "\x1b[1m"
)

func (o Options) paint(s, colour string) string {
	if !o.Color {
		return s
	}
	return colour + s + colReset
}

// WriteTTY renders the human report.
func (r *Report) WriteTTY(w io.Writer, o Options) error {
	g := o.glyphs()
	b := &strings.Builder{}

	fmt.Fprintf(b, "restored %s %s recipe %s %s run %s\n\n",
		r.Tool.Version, g.dot, r.Recipe.Name, g.dot, r.Run.ID)

	if r.Source.Kind != "" {
		fmt.Fprintf(b, "  %-10s %s  %s\n", "source", r.Source.Kind, r.Source.Repository)
	}
	if s := r.Source.Snapshot; s != nil {
		line := fmt.Sprintf("  %-10s %s  %s", "snapshot", s.ShortID, s.Time.Format("2006-01-02 15:04:05"))
		if s.Hostname != "" {
			line += "  host=" + s.Hostname
		}
		if len(s.Tags) > 0 {
			line += "  tags=[" + strings.Join(s.Tags, ",") + "]"
		}
		fmt.Fprintln(b, line)
	}
	r.writeInputs(b)
	fmt.Fprintln(b)

	for _, st := range r.Stages {
		status := "ok"
		colour := colGreen
		if st.Status != "ok" {
			status = "FAILED"
			colour = colRed
		}
		// The colour is applied after the padding, never before: escape bytes count
		// towards a width and would push every column along by five characters.
		line := fmt.Sprintf("  %-10s %s %8s", st.Name,
			o.paint(fmt.Sprintf("%-7s", status), colour), duration(st.DurationMS))
		if st.Note != "" {
			line += "   " + st.Note
		}
		fmt.Fprintln(b, strings.TrimRight(line, " "))
		if st.Error != "" {
			// The first line only. A stage row says which stage failed and roughly
			// why; the error block below says the whole thing, including the
			// instructions, and printing both in full says everything twice.
			headline := st.Error
			if i := strings.IndexByte(headline, '\n'); i >= 0 {
				headline = headline[:i]
			}
			for _, l := range wrap(headline, Width-14) {
				fmt.Fprintf(b, "  %-10s %s\n", "", l)
			}
		}
	}

	if len(r.Checks) > 0 {
		fmt.Fprintf(b, "\n  CHECKS\n")
		r.writeChecks(b, o, g)
	}

	fmt.Fprintln(b)
	verdict := r.Verdict
	colour := colGreen
	if r.Verdict != VerdictPass {
		verdict = "RESTORE UNUSABLE"
		colour = colRed
	}
	if r.Verdict == VerdictError {
		verdict = "ERROR"
	}
	summary := fmt.Sprintf("  %s  %d/%d checks", o.paint(verdict, colour+colBold),
		r.Summary.ChecksPassed, r.Summary.ChecksTotal)
	teardown := "teardown ok"
	if !r.Run.WorkspaceRemoved {
		teardown = "workspace kept"
	}
	fmt.Fprintf(b, "%s  %s  total %s  %s  %s\n", summary, g.dot,
		duration(r.Run.DurationMS), g.dot, teardown)

	if r.Error != "" {
		fmt.Fprintf(b, "\n  %s\n", r.Error)
	}

	if r.Hint != nil {
		r.writeHint(b, o)
	}

	if len(r.Warnings) > 0 {
		fmt.Fprintln(b)
		for _, wr := range r.Warnings {
			fmt.Fprintf(b, "  warning: %s (%s)\n", wr.Detail, wr.Code)
		}
	}

	switch r.Verdict {
	case VerdictPass:
		fmt.Fprintf(b, "\nThis backup boots.\n")
	case VerdictUnusable:
		fmt.Fprintf(b, "\n  Service logs from the failure window are in the JSON report (--report).\n")
		fmt.Fprintf(b, "  Re-run with --keep to keep the stack up and poke at it yourself.\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func (r *Report) writeInputs(b *strings.Builder) {
	if len(r.Inputs) == 0 {
		return
	}
	nameW, pathW := 0, 0
	for _, in := range r.Inputs {
		nameW = max(nameW, len(in.Name))
		pathW = max(pathW, len(in.BackupPath))
	}
	label := "inputs"
	for _, in := range r.Inputs {
		detail := fmt.Sprintf("%9s  %s", humanBytes(in.Bytes), fileCount(in))
		fmt.Fprintf(b, "  %-10s %-*s  %-*s  %s\n", label, nameW, in.Name, pathW, in.BackupPath, detail)
		label = ""
	}
}

func fileCount(in Input) string {
	switch in.DetectedFormat {
	case "plain":
		return "plain SQL"
	case "custom":
		return "custom-format dump"
	case "empty":
		return "empty"
	}
	if in.Files == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%s files", thousands(int64(in.Files)))
}

func (r *Report) writeChecks(b *strings.Builder, o Options, g glyphs) {
	idW := 0
	for _, c := range r.Checks {
		idW = max(idW, len(c.ID))
	}
	if idW > 22 {
		idW = 22
	}
	titleCol := 2 + 1 + 2 + idW + 2
	titleW := Width - titleCol - 8
	if titleW < 20 {
		titleW = 20
	}

	for _, c := range r.Checks {
		mark := o.paint(g.pass, colGreen)
		if c.Status != "pass" {
			mark = o.paint(g.fail, colRed)
		}
		text := c.Title
		if c.Observed.Summary != "" && c.Status == "pass" {
			text += "  " + g.arrow + "  " + c.Observed.Summary
		}
		lines := wrap(text, titleW)
		for i, l := range lines {
			switch i {
			case 0:
				fmt.Fprintf(b, "  %s  %-*s  %-*s %7s\n", mark, idW, c.ID, titleW, l, duration(c.DurationMS))
			default:
				fmt.Fprintf(b, "%s%s\n", strings.Repeat(" ", titleCol), l)
			}
		}
		if c.Status == "pass" {
			continue
		}
		indent := strings.Repeat(" ", titleCol+2)
		if c.Query != "" {
			fmt.Fprintf(b, "%squery   %s\n", indent, c.Query)
		}
		for _, f := range c.Failures {
			writeField(b, indent, "expect", f.Expect)
			writeField(b, indent, "got", f.Got)
		}
	}
}

func writeField(b *strings.Builder, indent, label, value string) {
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	for i, l := range lines {
		if i == 0 {
			fmt.Fprintf(b, "%s%-7s %s\n", indent, label, l)
			continue
		}
		fmt.Fprintf(b, "%s%-7s %s\n", indent, "", l)
	}
}

func (r *Report) writeHint(b *strings.Builder, o Options) {
	fmt.Fprintf(b, "\n  %s\n", o.paint("LIKELY CAUSE", colBold))
	for _, l := range wrap(r.Hint.Title, Width-6) {
		fmt.Fprintf(b, "    %s\n", l)
	}
	fmt.Fprintln(b)
	for _, para := range strings.Split(strings.TrimSpace(r.Hint.Text), "\n\n") {
		for _, l := range wrap(strings.Join(strings.Fields(para), " "), Width-6) {
			fmt.Fprintf(b, "    %s\n", l)
		}
	}
	if len(r.Hint.Commands) > 0 {
		fmt.Fprintln(b)
		for _, c := range r.Hint.Commands {
			fmt.Fprintf(b, "      %s\n", c)
		}
	}
	fmt.Fprintf(b, "%*s(hint: %s)\n", Width-len(r.Hint.ID)-9, "", r.Hint.ID)
}

// wrap breaks text at spaces to at most w columns.
func wrap(s string, w int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, word := range words[1:] {
		if len(cur)+1+len(word) > w {
			lines = append(lines, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	return append(lines, cur)
}

// duration renders a millisecond count the way the report shows it.
func duration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// humanBytes renders a size in binary units, the way du -h does.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit && exp < 4; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTP"[exp])
}

// thousands groups an integer so a file count is readable at a glance.
func thousands(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []string
	for len(s) > 3 {
		out = append([]string{s[len(s)-3:]}, out...)
		s = s[:len(s)-3]
	}
	out = append([]string{s}, out...)
	joined := strings.Join(out, ",")
	if neg {
		return "-" + joined
	}
	return joined
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
