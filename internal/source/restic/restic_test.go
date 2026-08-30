package restic

import (
	"strings"
	"testing"
)

// recorded is real `restic snapshots --json` output, trimmed to the fields drillback
// reads. Snapshot selection is tested against this rather than against a live
// repository, so the test costs nothing and cannot flake.
const recorded = `[
  {"time":"2026-09-11T02:14:07.1Z","tree":"aa","paths":["/srv/gitea"],
   "hostname":"hypervisor","username":"root","tags":["gitea"],
   "id":"4a7f1c2e5d6b7890abcdef0123456789abcdef0123456789abcdef0123456789","short_id":"4a7f1c2e"},
  {"time":"2026-09-13T02:14:07.1Z","tree":"bb","paths":["/srv/gitea"],
   "hostname":"hypervisor","username":"root","tags":["gitea","nightly"],
   "id":"9c1e77b0aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","short_id":"9c1e77b0"},
  {"time":"2026-09-13T02:14:07.1Z","tree":"cc","paths":["/srv/gitea"],
   "hostname":"laptop","username":"root","tags":[],
   "id":"1b2c3d4e5555555555555555555555555555555555555555555555555555555","short_id":"1b2c3d4e"},
  {"time":"2026-09-12T02:14:07.1Z","tree":"dd","paths":["/srv/vaultwarden"],
   "hostname":"hypervisor","username":"root","tags":["vaultwarden"],
   "id":"9c1e88ffbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","short_id":"9c1e88ff"}
]`

func snapshots(t *testing.T) []snapshotAlias {
	t.Helper()
	s, err := ParseSnapshots([]byte(recorded))
	if err != nil {
		t.Fatalf("parsing the recorded output: %v", err)
	}
	return s
}

func TestParseSnapshots(t *testing.T) {
	s := snapshots(t)
	if len(s) != 4 {
		t.Fatalf("parsed %d snapshots, want 4", len(s))
	}
	if s[0].ShortID != "4a7f1c2e" {
		t.Errorf("short id = %q", s[0].ShortID)
	}
	if s[1].Tags[1] != "nightly" {
		t.Errorf("tags = %v", s[1].Tags)
	}
	if got := s[0].Time.Format("2006-01-02"); got != "2026-09-11" {
		t.Errorf("time = %q", got)
	}
}

func TestSelectLatest(t *testing.T) {
	// Two snapshots share the newest timestamp, so the tie is broken by id and the
	// choice is the same on every run.
	got, err := Select(snapshots(t), "latest")
	if err != nil {
		t.Fatal(err)
	}
	if got.ShortID != "9c1e77b0" {
		t.Errorf("latest = %q, want 9c1e77b0 (the newer time, and the higher id of the two)", got.ShortID)
	}
	if got.SelectedBy != "latest" {
		t.Errorf("selected_by = %q", got.SelectedBy)
	}

	// An empty spec means the same thing as "latest".
	same, err := Select(snapshots(t), "")
	if err != nil || same.ID != got.ID {
		t.Errorf("an empty snapshot spec must mean latest, got %v %v", same, err)
	}
}

func TestSelectExplicit(t *testing.T) {
	cases := []struct{ spec, want string }{
		{"4a7f1c2e", "4a7f1c2e"},
		{"4A7F1C2E", "4a7f1c2e"},
		{"1b2c", "1b2c3d4e"},
		{"4a7f1c2e5d6b7890abcdef0123456789abcdef0123456789abcdef0123456789", "4a7f1c2e"},
	}
	for _, tc := range cases {
		got, err := Select(snapshots(t), tc.spec)
		if err != nil {
			t.Errorf("Select(%q): %v", tc.spec, err)
			continue
		}
		if got.ShortID != tc.want {
			t.Errorf("Select(%q) = %q, want %q", tc.spec, got.ShortID, tc.want)
		}
		if got.SelectedBy != "explicit" {
			t.Errorf("Select(%q) selected_by = %q", tc.spec, got.SelectedBy)
		}
	}
}

func TestSelectRejects(t *testing.T) {
	cases := []struct{ name, spec, want string }{
		{"an id that is not there", "deadbeef", "no snapshot with id"},
		{"an ambiguous prefix", "9c1e", "ambiguous"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Select(snapshots(t), tc.spec)
			if err == nil {
				t.Fatal("this selection must fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}

	if _, err := Select(nil, "latest"); err == nil {
		t.Error("an empty repository must be an error, not a nil snapshot")
	}
}

func TestLocate(t *testing.T) {
	cases := []struct{ dest, path, wantSuffix string }{
		{"/ws/restore", "/srv/gitea/data", "srv/gitea/data"},
		{"/ws/restore", "/srv/gitea/db.sql", "srv/gitea/db.sql"},
		{"/ws/restore", "/srv//gitea/", "srv/gitea"},
	}
	for _, tc := range cases {
		got := Locate(tc.dest, tc.path)
		if !strings.HasSuffix(filepathSlash(got), tc.wantSuffix) {
			t.Errorf("Locate(%q, %q) = %q, want it to end in %q", tc.dest, tc.path, got, tc.wantSuffix)
		}
	}
}
