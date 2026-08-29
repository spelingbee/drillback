package check

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spelingbee/restored/internal/recipe"
)

// Observation is what a check saw. Only the fields a kind produces are populated, and
// only populated fields reach the JSON report.
type Observation struct {
	Status    *int   `json:"status,omitempty"`
	BodyBytes *int   `json:"body_bytes,omitempty"`
	Matched   *bool  `json:"matched,omitempty"`
	Value     string `json:"value,omitempty"`
	Rows      *int   `json:"rows,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Count     *int   `json:"count,omitempty"`
	Entries   *int   `json:"entries,omitempty"`
	Bytes     *int64 `json:"bytes,omitempty"`
	Exists    *bool  `json:"exists,omitempty"`
	IsDir     *bool  `json:"is_dir,omitempty"`
	Error     string `json:"error,omitempty"`

	// Not serialised: the raw material the expect keys and the hint matcher read.
	Body    string `json:"-"`
	Stdout  string `json:"-"`
	Stderr  string `json:"-"`
	Summary string `json:"-"`
}

// Failure is one unmet expectation, in the "expect / got" shape the report prints.
type Failure struct {
	Expect string `json:"expect"`
	Got    string `json:"got"`
}

// Evaluate applies the expect vocabulary to an observation. Every key is checked, so a
// report shows every way a check missed rather than only the first.
func Evaluate(kind string, e recipe.Expect, o *Observation) []Failure {
	var f []Failure
	add := func(expect, got string) { f = append(f, Failure{Expect: expect, Got: got}) }

	if o.Error != "" {
		add(describeExpect(e), o.Error)
		return f
	}

	if e.Status != nil {
		if o.Status == nil || *o.Status != *e.Status {
			add(fmt.Sprintf("status: %d", *e.Status), fmt.Sprintf("status: %s", intOrDash(o.Status)))
		}
	}
	if len(e.StatusIn) > 0 {
		ok := false
		for _, s := range e.StatusIn {
			if o.Status != nil && *o.Status == s {
				ok = true
			}
		}
		if !ok {
			add(fmt.Sprintf("status_in: %v", e.StatusIn), fmt.Sprintf("status: %s", intOrDash(o.Status)))
		}
	}

	subject := o.Body
	if kind == "exec" {
		subject = o.Stdout
	}
	if e.BodyMatches != "" {
		if !matches(e.BodyMatches, subject) {
			add("body_matches: "+e.BodyMatches, snippet(subject))
		} else {
			t := true
			o.Matched = &t
		}
	}
	if e.BodyNotMatches != "" && matches(e.BodyNotMatches, subject) {
		add("body_not_matches: "+e.BodyNotMatches, snippet(subject))
	}

	if e.JSONPath != "" {
		var doc any
		if err := json.Unmarshal([]byte(o.Body), &doc); err != nil {
			add("json_path: "+e.JSONPath, "the response body is not JSON: "+snippet(o.Body))
		} else if v, err := Lookup(doc, e.JSONPath); err != nil {
			add("json_path: "+e.JSONPath, err.Error())
		} else {
			o.Value = describe(v)
			if e.JSONPathEquals != nil {
				if s, ok := v.(string); !ok || s != *e.JSONPathEquals {
					add(fmt.Sprintf("json_path_equals: %s", *e.JSONPathEquals), describe(v))
				}
			}
			if e.JSONPathIntMin != nil {
				n, ok := asInt(v)
				if !ok || n < *e.JSONPathIntMin {
					add(fmt.Sprintf("json_path_int_min: %d", *e.JSONPathIntMin), describe(v))
				}
			}
			if e.JSONPathLenMin != nil {
				n, ok := lengthOf(v)
				if !ok {
					add(fmt.Sprintf("json_path_len_min: %d", *e.JSONPathLenMin), describe(v)+" has no length")
				} else {
					o.Count = &n
					o.Summary = fmt.Sprintf("%d item%s", n, plural(n))
					if n < *e.JSONPathLenMin {
						add(fmt.Sprintf("json_path_len_min: %d", *e.JSONPathLenMin), fmt.Sprintf("%d", n))
					}
				}
			}
		}
	}

	if kind == "exec" {
		want := 0
		if e.ExitCode != nil {
			want = *e.ExitCode
		}
		if o.ExitCode == nil || *o.ExitCode != want {
			add(fmt.Sprintf("exit_code: %d", want), fmt.Sprintf("exit_code: %s", intOrDash(o.ExitCode)))
		}
	}
	if e.StdoutMatches != "" && !matches(e.StdoutMatches, o.Stdout) {
		add("stdout_matches: "+e.StdoutMatches, snippet(o.Stdout))
	}
	if e.StderrMatches != "" && !matches(e.StderrMatches, o.Stderr) {
		add("stderr_matches: "+e.StderrMatches, snippet(o.Stderr))
	}

	if e.ScalarEquals != nil && o.Value != *e.ScalarEquals {
		add("scalar_equals: "+*e.ScalarEquals, quoteEmpty(o.Value))
	}
	if e.ScalarIntMin != nil {
		n, err := strconv.Atoi(strings.TrimSpace(o.Value))
		if err != nil {
			add(fmt.Sprintf("scalar_int_min: %d", *e.ScalarIntMin), quoteEmpty(o.Value))
		} else if n < *e.ScalarIntMin {
			add(fmt.Sprintf("scalar_int_min: %d", *e.ScalarIntMin), strconv.Itoa(n))
		}
	}
	if e.ScalarIntMax != nil {
		n, err := strconv.Atoi(strings.TrimSpace(o.Value))
		if err != nil {
			add(fmt.Sprintf("scalar_int_max: %d", *e.ScalarIntMax), quoteEmpty(o.Value))
		} else if n > *e.ScalarIntMax {
			add(fmt.Sprintf("scalar_int_max: %d", *e.ScalarIntMax), strconv.Itoa(n))
		}
	}
	if e.RowsMin != nil {
		if o.Rows == nil || *o.Rows < *e.RowsMin {
			add(fmt.Sprintf("rows_min: %d", *e.RowsMin), fmt.Sprintf("rows: %s", intOrDash(o.Rows)))
		}
	}
	if e.RowsMax != nil {
		if o.Rows == nil || *o.Rows > *e.RowsMax {
			add(fmt.Sprintf("rows_max: %d", *e.RowsMax), fmt.Sprintf("rows: %s", intOrDash(o.Rows)))
		}
	}

	if e.Exists != nil {
		got := o.Exists != nil && *o.Exists
		if got != *e.Exists {
			add(fmt.Sprintf("exists: %t", *e.Exists), fmt.Sprintf("exists: %t", got))
		}
	}
	if e.IsDir != nil {
		got := o.IsDir != nil && *o.IsDir
		if got != *e.IsDir {
			add(fmt.Sprintf("is_dir: %t", *e.IsDir), fmt.Sprintf("is_dir: %t", got))
		}
	}
	if e.NotEmpty != nil {
		got := o.Entries != nil && *o.Entries > 0
		if got != *e.NotEmpty {
			add(fmt.Sprintf("not_empty: %t", *e.NotEmpty), fmt.Sprintf("entries: %s", intOrDash(o.Entries)))
		}
	}
	if e.SizeMin != nil {
		if o.Bytes == nil || *o.Bytes < *e.SizeMin {
			add(fmt.Sprintf("size_min: %d", *e.SizeMin), fmt.Sprintf("bytes: %s", int64OrDash(o.Bytes)))
		}
	}
	if e.GlobMinCount != nil {
		if o.Count == nil || *o.Count < *e.GlobMinCount {
			add(fmt.Sprintf("glob_min_count: %d for %s", *e.GlobMinCount, e.Glob),
				fmt.Sprintf("%s matches", intOrDash(o.Count)))
		}
	}
	return f
}

// describeExpect renders the whole expect block for the case where the check could not
// run at all and there is nothing to compare against.
func describeExpect(e recipe.Expect) string {
	b, err := json.Marshal(e)
	if err != nil {
		return "the recipe's expectations"
	}
	return strings.Trim(string(b), "{}")
}

func matches(pattern, subject string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(subject)
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), t == float64(int64(t))
	case string:
		n, err := strconv.Atoi(t)
		return n, err == nil
	default:
		return 0, false
	}
}

func intOrDash(p *int) string {
	if p == nil {
		return "-"
	}
	return strconv.Itoa(*p)
}

func int64OrDash(p *int64) string {
	if p == nil {
		return "-"
	}
	return strconv.FormatInt(*p, 10)
}

func quoteEmpty(s string) string {
	if s == "" {
		return `""`
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func snippet(s string) string {
	s = strings.TrimSpace(s)
	const max = 300
	if len(s) > max {
		return s[:max] + "..."
	}
	if s == "" {
		return `""`
	}
	return s
}
