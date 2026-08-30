package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewCreatesTheTree(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "k7m2q9xf")
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Base(ws.Root); got != "restored-k7m2q9xf" {
		t.Errorf("workspace directory = %q", got)
	}
	if got := ws.ProjectName(); got != "restored-k7m2q9xf" {
		t.Errorf("compose project = %q", got)
	}
	for _, d := range []string{ws.InputsDir(), ws.RestoreDir(), ws.LogsDir(), ws.TestAssetsDir(), ws.ExportDir()} {
		if info, err := os.Stat(d); err != nil || !info.IsDir() {
			t.Errorf("%s is not a directory: %v", d, err)
		}
	}

	if err := ws.Remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ws.Root); !os.IsNotExist(err) {
		t.Error("the workspace survived Remove")
	}
	// Teardown runs on several paths and must never object to having already run.
	if err := ws.Remove(); err != nil {
		t.Errorf("Remove is not idempotent: %v", err)
	}
}

func TestRunIDIsUsableEverywhere(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id, err := NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate run id %q after %d draws", id, i)
		}
		seen[id] = true
		// The id names a directory, a compose project, and a docker label, so it has
		// to survive all three without quoting.
		for _, c := range id {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz234567", c) {
				t.Fatalf("run id %q contains %q", id, c)
			}
		}
	}
}

func TestContains(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	inside := []string{
		ws.Root,
		filepath.Join(ws.Root, "inputs"),
		filepath.Join(ws.Root, "inputs", "data", "deep", "file"),
	}
	for _, p := range inside {
		if !ws.Contains(p) {
			t.Errorf("Contains(%q) = false, want true", p)
		}
	}
	outside := []string{
		filepath.Dir(ws.Root),
		filepath.Join(ws.Root, "..", "elsewhere"),
		filepath.Join(ws.Root, "inputs", "..", "..", "etc"),
	}
	for _, p := range outside {
		if ws.Contains(p) {
			t.Errorf("Contains(%q) = true, want false", p)
		}
	}
}

// A backup containing an /etc/shadow symlink is a real thing, and it must not become
// a read of the host's /etc/shadow from inside a container.
func TestSanitiseNeutralisesEscapingSymlinks(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(ws.InputsDir(), "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(data, "real.txt")
	if err := os.WriteFile(real, []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}

	escaping := filepath.Join(data, "shadow")
	if err := os.Symlink(filepath.Join(string(filepath.Separator), "etc", "shadow"), escaping); err != nil {
		t.Skipf("this platform will not create symlinks for this user: %v", err)
	}
	internal := filepath.Join(data, "inside")
	if err := os.Symlink(real, internal); err != nil {
		t.Fatal(err)
	}

	warnings, err := ws.Sanitise(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1: %v", len(warnings), warnings)
	}
	if warnings[0].Code != "symlink_escaped_workspace" {
		t.Errorf("warning code = %q", warnings[0].Code)
	}
	if !strings.Contains(warnings[0].Detail, "shadow") {
		t.Errorf("warning detail %q does not name the link", warnings[0].Detail)
	}

	info, err := os.Lstat(escaping)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("the escaping symlink is still a symlink")
	}
	if info.Size() != 0 {
		t.Errorf("the neutralised link is %d bytes, want 0", info.Size())
	}

	// A link that stays inside the workspace is left alone: it is part of the data.
	if _, err := os.Lstat(internal); err != nil {
		t.Errorf("the internal symlink was removed: %v", err)
	}
	if b, err := os.ReadFile(real); err != nil || string(b) != "kept" {
		t.Errorf("the real file was disturbed: %q %v", b, err)
	}
}

func TestSanitiseRefusesToLeaveTheWorkspace(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Sanitise(filepath.Dir(ws.Root)); err == nil {
		t.Fatal("sanitising a path outside the workspace must be refused")
	}
}

func TestMeasure(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b"), make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Measure(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Files != 2 || st.Bytes != 300 {
		t.Errorf("Measure = %+v, want 2 files and 300 bytes", st)
	}

	// A restore that produced nothing has to be visible in the report even when every
	// check somehow passes.
	empty, err := Measure(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if empty.Bytes != 200 {
		t.Errorf("Measure of a subdirectory = %+v", empty)
	}
}

func TestCopyTree(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "f"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "copy")
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "sub", "f"))
	if err != nil || string(b) != "data" {
		t.Errorf("copied file = %q %v", b, err)
	}
}

// A restored tree carries the ownership and the modes of the machine the backup came
// from. Neither means anything inside a fresh container, and a 0700 tree is one the
// application cannot read. See ADR-055.
func TestRelaxOpensARestoredTree(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "relax")
	if err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(ws.InputsDir(), "data")
	nested := filepath.Join(tree, "git", "repositories")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "HEAD")
	if err := os.WriteFile(file, []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ws.Relax(tree); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		// Windows carries no POSIX mode through a chmod, and the daemon does not
		// enforce one on a bind mount; there is nothing to assert but the absence of
		// an error.
		return
	}
	for path, want := range map[string]os.FileMode{
		tree:   relaxDirMode,
		nested: relaxDirMode,
		file:   relaxFileMode,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s: mode %04o, want %04o", path, got, want)
		}
	}
}

// The workspace itself stays shut, which is what makes the modes above safe: the
// files are permissive inside a directory nobody else can traverse. See ADR-054.
func TestWorkspaceRootIsPrivate(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "private")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(ws.Root)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("workspace root mode %04o, want 0700", got)
	}
}

func TestRelaxRefusesToLeaveTheWorkspace(t *testing.T) {
	ws, err := NewWithID(t.TempDir(), "escape")
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := ws.Relax(outside); err == nil {
		t.Fatalf("Relax(%q) must be refused: it is outside the workspace", outside)
	}
}
