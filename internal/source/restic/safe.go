package restic

import (
	"net/url"
	"strings"
)

// SafeRepository is a restic repository string with any password taken out of it.
//
// restic's REST, S3, Azure, B2, Swift and rclone backends all accept credentials
// inside the repository string - `rest:https://user:password@host:8000/` is a
// documented and common configuration - and the repository string goes into the
// report, onto the terminal, and into the debug log. SPEC.md section 9.3 has always
// promised the URL is "scrubbed of any user:password@ userinfo"; until the session 4
// security review (docs/review/security.md SEC-03) nothing did the scrubbing.
//
// The user name is kept, because it is often the only way to tell two repositories
// apart in a report, and it is not the secret.
func SafeRepository(repo string) string {
	if repo == "" {
		return ""
	}
	// A restic repository is `<backend>:<rest>`, and only some backends put a URL in
	// the second half. A bare Windows path (C:\backups) and a bare POSIX path both
	// have to come through untouched.
	prefix, rest, ok := cutBackend(repo)
	if !ok {
		return repo
	}
	u, err := url.Parse(rest)
	if err != nil || u.User == nil {
		// Not a URL, or no userinfo. `sftp:user@host:/path` is the common shape here
		// and carries no password, so it is left alone.
		return repo
	}
	if _, hasPassword := u.User.Password(); !hasPassword {
		return repo
	}
	u.User = url.User(u.User.Username())
	return prefix + ":" + u.String()
}

// cutBackend splits `rest:https://...` into `rest` and `https://...`. It returns
// false for anything that is not one of the backends whose second half is a URL,
// which is what keeps local paths - including Windows drive letters - untouched.
func cutBackend(repo string) (prefix, rest string, ok bool) {
	i := strings.Index(repo, ":")
	if i <= 0 {
		return "", "", false
	}
	switch repo[:i] {
	case "rest", "s3", "azure", "b2", "swift", "gs", "rclone":
		return repo[:i], repo[i+1:], true
	}
	return "", "", false
}
