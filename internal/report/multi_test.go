package report

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func sampleRun(verdict string, ms int64, passed, total int, errStr string) *Report {
	r := &Report{
		SchemaVersion: SchemaVersion,
		Tool:          Tool{Name: "restored", Version: "0.1.0-test"},
		Verdict:       verdict,
		Run:           Run{ID: "testrun", DurationMS: ms, WorkspaceRemoved: true},
		Summary:       Summary{ChecksTotal: total, ChecksPassed: passed, ChecksFailed: total - passed},
		Error:         errStr,
	}
	switch verdict {
	case VerdictPass:
		r.ExitCode = 0
	case VerdictUnusable:
		r.ExitCode = 1
	default:
		r.ExitCode = 2
	}
	return r
}

func sampleMulti() *Multi {
	m := NewMulti()
	// 12m 04s on purpose: the [10m, 1h) band renders seven columns, and a golden
	// with only sub-10m samples could not catch the alignment regressing.
	m.Add("gitea", sampleRun(VerdictPass, 724_000, 5, 5, ""))
	m.Add("vaultwarden", sampleRun(VerdictUnusable, 26_900, 3, 5, ""))
	m.Add("paperless", sampleRun(VerdictError, 1_200, 0, 0,
		"no snapshot with tag \"paperless\" in the repository\nsecond line the summary must not show"))
	return m
}

func TestMultiAggregatesTheWorstOutcome(t *testing.T) {
	m := NewMulti()
	if m.ExitCode != 0 {
		t.Fatalf("empty exit = %d", m.ExitCode)
	}
	m.Add("a", sampleRun(VerdictPass, 10, 1, 1, ""))
	if m.ExitCode != 0 {
		t.Errorf("after a PASS, exit = %d, want 0", m.ExitCode)
	}
	m.Add("b", sampleRun(VerdictUnusable, 10, 0, 1, ""))
	if m.ExitCode != 1 {
		t.Errorf("after an UNUSABLE, exit = %d, want 1", m.ExitCode)
	}
	m.Add("c", sampleRun(VerdictError, 10, 0, 0, "boom"))
	if m.ExitCode != 2 {
		t.Errorf("after an ERROR, exit = %d, want 2", m.ExitCode)
	}
	// An error is the worst outcome; a later unusable must not demote it.
	m.Add("d", sampleRun(VerdictUnusable, 10, 0, 1, ""))
	if m.ExitCode != 2 {
		t.Errorf("an ERROR already seen, exit = %d, want it to stay 2", m.ExitCode)
	}
	s := m.Summary
	if s.TargetsTotal != 4 || s.TargetsPassed != 1 || s.TargetsUnusable != 2 || s.TargetsErrored != 1 {
		t.Errorf("summary = %+v", s)
	}
	if s.DurationMS != 40 {
		t.Errorf("duration = %d, want the sum", s.DurationMS)
	}
}

func TestSkipRemainingMakesAnInterruptedSweepUnprovable(t *testing.T) {
	m := NewMulti()
	m.Add("a", sampleRun(VerdictPass, 10, 1, 1, ""))
	m.SkipRemaining(0)
	if m.ExitCode != 0 || m.Summary.TargetsSkipped != 0 {
		t.Errorf("a completed sweep must stay as its verdicts left it: exit=%d skipped=%d", m.ExitCode, m.Summary.TargetsSkipped)
	}
	m.SkipRemaining(3)
	if m.ExitCode != 2 {
		t.Errorf("exit = %d; skipped targets are unproven, and unproven is exit 2 (ADR-058)", m.ExitCode)
	}
	if m.Summary.TargetsSkipped != 3 {
		t.Errorf("skipped = %d", m.Summary.TargetsSkipped)
	}
}

func TestEmptyMultiMarshalsRunsAsAnArray(t *testing.T) {
	var buf bytes.Buffer
	if err := NewMulti().WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"runs": []`)) {
		t.Errorf("an empty sweep owes SPEC 5.2 an array, not null:\n%s", buf.String())
	}
}

func TestMultiJSONIsTheSingleDocumentPlusTarget(t *testing.T) {
	var buf bytes.Buffer
	if err := sampleMulti().WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		SchemaVersion int `json:"schema_version"`
		Runs          []map[string]any
		Summary       map[string]any
		ExitCode      int `json:"exit_code"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != SchemaVersion || doc.ExitCode != 2 {
		t.Errorf("schema_version=%d exit_code=%d", doc.SchemaVersion, doc.ExitCode)
	}
	if len(doc.Runs) != 3 {
		t.Fatalf("runs = %d", len(doc.Runs))
	}
	first := doc.Runs[0]
	if first["target"] != "gitea" {
		t.Errorf("runs[0].target = %v", first["target"])
	}
	// The element must be the single-run document, flattened - not nested under a key.
	for _, key := range []string{"verdict", "exit_code", "summary", "run", "tool"} {
		if _, ok := first[key]; !ok {
			t.Errorf("runs[0] is missing %q: the element must be exactly the single-run document", key)
		}
	}
	if doc.Summary["targets_errored"] != float64(1) {
		t.Errorf("summary = %v", doc.Summary)
	}
}

// SPEC.md section 10 asks for golden renderings of the multi-target shape in both
// colour and NO_COLOR. Rewrite with `go test ./internal/report -update`.
func TestMultiTTYGolden(t *testing.T) {
	interrupted := func() *Multi {
		m := sampleMulti()
		m.SkipRemaining(2)
		return m
	}
	cases := []struct {
		name  string
		multi *Multi
		opts  Options
	}{
		{"all-colour", sampleMulti(), Options{Color: true}},
		{"all-plain", sampleMulti(), Options{Color: false}},
		{"all-ascii", sampleMulti(), Options{Color: false, ASCII: true}},
		{"all-interrupted", interrupted(), Options{Color: false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := tc.multi.WriteTTY(&b, tc.opts); err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", tc.name+".txt")
			if *update {
				if err := os.WriteFile(golden, b.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run `go test ./internal/report -update` to create it)", err)
			}
			if !bytes.Equal(b.Bytes(), want) {
				t.Errorf("rendering drifted from %s:\n--- got ---\n%s\n--- want ---\n%s", golden, b.String(), want)
			}
		})
	}
}
