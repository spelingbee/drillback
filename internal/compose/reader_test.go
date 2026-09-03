package compose

import (
	"path/filepath"
	"testing"
)

// The listing is what the helper printed; the entries are what a check judges. The
// name is the last field so that a separator inside it does not split the line.
func TestParseListing(t *testing.T) {
	out := "directory|4096|/inputs/data\n" +
		"regular file|108544|/inputs/data/users/drilladmin/db.sqlite\n" +
		"regular empty file|0|/inputs/data/users/drilladmin/log|with|pipes.txt\n" +
		"directory|4096|/inputs/data/users\n" +
		"regular file|12|/inputs/other/escaped.txt\n" +
		"not a listing line\n" +
		"\n"
	got := parseListing(out, "/inputs/data")
	want := []Entry{
		{Rel: "", IsDir: true, Size: 4096},
		{Rel: "users/drilladmin/db.sqlite", Size: 108544},
		{Rel: "users/drilladmin/log|with|pipes.txt", Size: 0},
		{Rel: "users", IsDir: true, Size: 4096},
	}
	if len(got) != len(want) {
		t.Fatalf("parseListing returned %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if !hasSelf(got) {
		t.Error("the listing includes the path itself, and hasSelf did not see it")
	}
	if hasSelf(parseListing("regular file|1|/inputs/data/x\n", "/inputs/data")) {
		t.Error("a listing without the path itself must not count as having it")
	}
}

// A sibling of the listed path must not leak into its listing: /inputs/data-old is
// not under /inputs/data.
func TestParseListingDoesNotMatchSiblingPrefixes(t *testing.T) {
	got := parseListing("directory|4096|/inputs/data\nregular file|5|/inputs/data-old/x\n", "/inputs/data")
	if len(got) != 1 || got[0].Rel != "" {
		t.Fatalf("got %+v, want only the path itself", got)
	}
}

// The helper is bound to the inputs tree and nothing else, so every path it is asked
// about has to be inside that tree.
func TestContainerPath(t *testing.T) {
	inputs := filepath.Join(t.TempDir(), "ws", "inputs")
	r := &Reader{InputsDir: inputs}
	cases := []struct {
		host string
		want string
		ok   bool
	}{
		{filepath.Join(inputs, "data", "db.sqlite"), "/inputs/data/db.sqlite", true},
		{inputs, "/inputs", true},
		{filepath.Join(inputs, "..", "compose.yaml"), "", false},
		{filepath.Join(inputs, "..", "inputs-old", "x"), "", false},
		{filepath.Dir(inputs), "", false},
	}
	for _, tc := range cases {
		got, err := r.containerPath(tc.host)
		if tc.ok && (err != nil || got != tc.want) {
			t.Errorf("containerPath(%q) = %q, %v; want %q", tc.host, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Errorf("containerPath(%q) = %q, want a refusal", tc.host, got)
		}
	}
}
