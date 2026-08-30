package restic

import "testing"

// SEC-03. Every one of these repository strings is a documented restic form, and the
// ones with a password in them used to reach the report, the terminal and the debug
// log verbatim.
func TestSafeRepository(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"rest:https://user:hunter2@backup.example:8000/", "rest:https://user@backup.example:8000/"},
		{"rest:http://user:hunter2@10.0.0.5:8000/repo", "rest:http://user@10.0.0.5:8000/repo"},
		{"s3:https://key:secret@s3.example.com/bucket", "s3:https://key@s3.example.com/bucket"},
		{"b2:https://id:key@api.backblazeb2.com/b", "b2:https://id@api.backblazeb2.com/b"},

		// No password, nothing to take out. A user name is not a secret, and it is
		// often the only way to tell two repositories apart in a report.
		{"rest:https://user@backup.example:8000/", "rest:https://user@backup.example:8000/"},
		{"rest:https://backup.example:8000/", "rest:https://backup.example:8000/"},
		{"sftp:backupuser@nas.local:/srv/restic", "sftp:backupuser@nas.local:/srv/restic"},

		// Local paths, which are the common case and must come through untouched -
		// including a Windows drive letter, whose colon looks like a backend prefix.
		{"/srv/backups/restic", "/srv/backups/restic"},
		{"C:/backups/restic", "C:/backups/restic"},
		{"", ""},
	}
	for _, c := range cases {
		if got := SafeRepository(c.in); got != c.want {
			t.Errorf("SafeRepository(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

// The property that matters, stated on its own: whatever the input, the password does
// not survive.
func TestSafeRepositoryNeverKeepsThePassword(t *testing.T) {
	for _, repo := range []string{
		"rest:https://user:hunter2@backup.example:8000/",
		"s3:https://key:hunter2@s3.example.com/bucket",
		"azure:https://account:hunter2@blob.example.com/c",
		"swift:https://u:hunter2@swift.example.com/c",
		"gs:https://u:hunter2@storage.example.com/b",
		"rclone:https://u:hunter2@rclone.example.com/r",
	} {
		if got := SafeRepository(repo); contains(got, "hunter2") {
			t.Errorf("SafeRepository(%q) kept the password: %q", repo, got)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
