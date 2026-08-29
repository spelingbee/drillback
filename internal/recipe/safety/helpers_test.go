package safety

import "os"

// writeFile is a thin helper so the tests can put a fixture on disk without importing
// os in every file.
func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}
