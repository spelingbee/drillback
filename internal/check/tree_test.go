package check

import (
	"testing"

	"github.com/spelingbee/drillback/internal/compose"
)

// A listing has to reach exactly as deep as the glob does: one level for the direct
// entries `not_empty` counts, one per segment for a pattern.
func TestGlobDepth(t *testing.T) {
	cases := map[string]int{
		"":                 1,
		"*.sqlite":         1,
		"repos/*/*.git":    3,
		"/documents/*.pdf": 2,
		"a/b/c/d/":         4,
	}
	for glob, want := range cases {
		if got := globDepth(glob); got != want {
			t.Errorf("globDepth(%q) = %d, want %d", glob, got, want)
		}
	}
}

// observeTree is the host-side judgement the `file` kind used to make with os.Stat,
// os.ReadDir and filepath.Glob, now made from what the daemon listed.
func TestObserveTree(t *testing.T) {
	tree := []compose.Entry{
		{Rel: "", IsDir: true, Size: 4096},
		{Rel: "config.php", Size: 1234},
		{Rel: ".htaccess", Size: 10},
		{Rel: "repos", IsDir: true, Size: 4096},
		{Rel: "repos/alice", IsDir: true, Size: 4096},
		{Rel: "repos/alice/site.git", IsDir: true, Size: 4096},
		{Rel: "repos/bob", IsDir: true, Size: 4096},
		{Rel: "repos/bob/notes.git", IsDir: true, Size: 4096},
		{Rel: "repos/bob/README", Size: 3},
	}

	t.Run("a directory counts its direct entries", func(t *testing.T) {
		obs := observeTree(tree, true, "")
		if obs.Exists == nil || !*obs.Exists || obs.IsDir == nil || !*obs.IsDir {
			t.Fatalf("got %+v, want an existing directory", obs)
		}
		if obs.Entries == nil || *obs.Entries != 3 {
			t.Errorf("entries = %v, want 3 (config.php, .htaccess, repos)", obs.Entries)
		}
		if obs.Bytes != nil || obs.Count != nil {
			t.Errorf("a directory without a glob has no bytes and no count: %+v", obs)
		}
	})

	t.Run("a glob counts matches at its own depth only", func(t *testing.T) {
		obs := observeTree(tree, true, "repos/*/*.git")
		if obs.Count == nil || *obs.Count != 2 {
			t.Errorf("count = %v, want 2", obs.Count)
		}
		if obs.Summary != "2 matches for repos/*/*.git" {
			t.Errorf("summary = %q", obs.Summary)
		}
		obs = observeTree(tree, true, "*")
		if obs.Count == nil || *obs.Count != 3 {
			t.Errorf("count for * = %v, want 3: a dotfile matches, as it did for filepath.Glob", obs.Count)
		}
		obs = observeTree(tree, true, "nothing-*")
		if obs.Count == nil || *obs.Count != 0 || obs.Summary != "0 matches for nothing-*" {
			t.Errorf("got %+v, want zero matches", obs)
		}
	})

	t.Run("a file has a size and no entries", func(t *testing.T) {
		obs := observeTree([]compose.Entry{{Rel: "", Size: 108544}}, true, "")
		if obs.IsDir == nil || *obs.IsDir {
			t.Fatalf("got %+v, want a file", obs)
		}
		if obs.Bytes == nil || *obs.Bytes != 108544 || obs.Summary != "108544 bytes" {
			t.Errorf("got %+v, want 108544 bytes", obs)
		}
		if obs.Entries != nil {
			t.Errorf("a file has no entries: %+v", obs)
		}
	})

	t.Run("a missing path exists false and nothing else", func(t *testing.T) {
		obs := observeTree(nil, false, "*.pdf")
		if obs.Exists == nil || *obs.Exists {
			t.Fatalf("got %+v, want exists=false", obs)
		}
		if obs.IsDir != nil || obs.Count != nil || obs.Error != "" {
			t.Errorf("a missing path reports nothing else: %+v", obs)
		}
	})

	t.Run("a bad pattern is an error, not a zero", func(t *testing.T) {
		obs := observeTree(tree, true, "[")
		if obs.Error == "" || obs.Count != nil {
			t.Errorf("got %+v, want an error and no count", obs)
		}
	})
}
