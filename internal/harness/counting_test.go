package harness

import (
	"testing"

	"github.com/spelingbee/drillback/internal/recipe"
)

func intp(v int) *int { return &v }

// MNT-05. When the empty stack refuses to start, stage A runs no check at all, so
// nothing has been tested against emptiness. A recipe whose checks are all
// `status: 200` and `exists: true` could then pass stage A by refusal, pass stage B on
// real data, go green in CI, and pass against a schema-only dump - the false PASS this
// tool exists to destroy, with a green badge on it. See ADR-064.
func TestCountingChecks(t *testing.T) {
	cases := map[string]struct {
		checks []*recipe.Check
		want   []string
	}{
		"nothing that counts": {
			checks: []*recipe.Check{
				{ID: "home", Expect: recipe.Expect{Status: intp(200)}},
				{ID: "dir", Expect: recipe.Expect{IsDir: boolp(true), Exists: boolp(true)}},
				{ID: "exit", Expect: recipe.Expect{ExitCode: intp(0)}},
			},
			want: nil,
		},
		"a row count": {
			checks: []*recipe.Check{
				{ID: "home", Expect: recipe.Expect{Status: intp(200)}},
				{ID: "users", Expect: recipe.Expect{ScalarIntMin: intp(1)}},
			},
			want: []string{"users"},
		},
		"a glob count": {
			checks: []*recipe.Check{{ID: "repos", Expect: recipe.Expect{GlobMinCount: intp(1)}}},
			want:   []string{"repos"},
		},
		"a JSON array length": {
			checks: []*recipe.Check{{ID: "api", Expect: recipe.Expect{JSONPathLenMin: intp(1)}}},
			want:   []string{"api"},
		},
		"a body that must match": {
			checks: []*recipe.Check{{ID: "page", Expect: recipe.Expect{BodyMatches: "welcome back"}}},
			want:   []string{"page"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := countingChecks(&recipe.Recipe{Checks: c.checks})
			if len(got) != len(c.want) {
				t.Fatalf("countingChecks = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("countingChecks = %v, want %v", got, c.want)
				}
			}
		})
	}
}

// The rejection has to say what to add, not just that something is missing: it is the
// only thing a contributor whose application refuses to boot empty will read.
func TestNoCountingCheckNamesTheVocabulary(t *testing.T) {
	for _, key := range []string{"scalar_int_min", "rows_min", "json_path_len_min", "glob_min_count"} {
		if !contains(NoCountingCheck, key) {
			t.Errorf("the rejection message does not name %s", key)
		}
	}
}

func boolp(v bool) *bool { return &v }

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
