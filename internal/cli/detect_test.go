package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/recipe/safety"
)

const sqliteCompose = `services:
  vaultwarden:
    image: vaultwarden/server:1.32.7-alpine
    ports:
      - "127.0.0.1:8080:80"
    environment:
      DATABASE_URL: /data/db.sqlite3
      ADMIN_TOKEN: ${VW_ADMIN_TOKEN}
    volumes:
      - ./vw-data:/data
`

func TestDetectComposeReadsARealFile(t *testing.T) {
	d, err := detectCompose(filepath.Join("..", "..", "testdata", "compose", "paperless.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if d.AppService != "webserver" {
		t.Errorf("app service = %q, want webserver", d.AppService)
	}
	// The container-side port is the one that exists on the run's internal network.
	// The published side is the only part the operator chose, so it is the part to
	// throw away.
	if d.AppPort != 8000 {
		t.Errorf("app port = %d, want 8000", d.AppPort)
	}
	if d.DB == nil || d.DB.Kind != "postgres-dump" || d.DB.Service != "db" {
		t.Fatalf("database = %+v, want a postgres-dump in service db", d.DB)
	}
	if d.DB.Name != "paperless" || d.DB.User != "paperless" {
		t.Errorf("database name/user = %q/%q, want paperless/paperless", d.DB.Name, d.DB.User)
	}
	// The operator's own ${PAPERLESS_DB_PASSWORD} is not a value restored can use.
	if d.DB.Password != "" {
		t.Errorf("a password read out of the operator's environment must not be carried over: %q", d.DB.Password)
	}

	var names []string
	for _, dir := range d.Dirs {
		names = append(names, dir.Name)
		if dir.Service != "webserver" {
			t.Errorf("input %q came from service %q: only the application's own state is an input", dir.Name, dir.Service)
		}
	}
	want := "data media export consume"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("dir inputs = %q, want %q", got, want)
	}
	if !d.Infra["db"] || !d.Infra["broker"] {
		t.Errorf("db and broker must be infrastructure, got %v", d.Infra)
	}
	if len(d.Notes) != 1 || !strings.Contains(d.Notes[0], "broker") {
		t.Errorf("want one note about the cache, got %v", d.Notes)
	}
}

func TestDetectComposeFindsASQLiteFileInTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte(sqliteCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := detectCompose(file)
	if err != nil {
		t.Fatal(err)
	}
	if d.DB == nil || d.DB.Kind != "sqlite" || d.DB.File != "/data/db.sqlite3" {
		t.Fatalf("database = %+v, want the sqlite file from DATABASE_URL", d.DB)
	}
	if d.AppPort != 80 {
		t.Errorf("app port = %d, want 80 from the container side of 127.0.0.1:8080:80", d.AppPort)
	}
	within, rel := sqliteWithin(d)
	if within != "data" || rel != "data/db.sqlite3" {
		t.Errorf("sqliteWithin = %q, %q; want the file located inside the data input", within, rel)
	}
}

// The whole point of --compose is that a contributor's first command succeeds. A
// proposal that does not validate is worse than no proposal.
func TestScaffoldFromComposeValidates(t *testing.T) {
	cases := map[string]string{
		"paperless-demo":   filepath.Join("..", "..", "testdata", "compose", "paperless.yml"),
		"vaultwarden-demo": "",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if src == "" {
				src = filepath.Join(t.TempDir(), "compose.yml")
				if err := os.WriteFile(src, []byte(sqliteCompose), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			d, err := detectCompose(src)
			if err != nil {
				t.Fatal(err)
			}
			files, err := scaffoldFromCompose(name, d)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for file, body := range files {
				if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			r, err := recipe.Load(dir)
			if err != nil {
				t.Fatalf("the proposed recipe does not validate: %v\n%s", err, files["recipe.yaml"])
			}
			composeRaw, err := r.ReadFile("compose.yaml")
			if err != nil {
				t.Fatal(err)
			}
			res, err := recipe.Resolve(r, recipe.Options{InputsDir: "/ws/inputs", RunID: "detect"})
			if err != nil {
				t.Fatalf("the proposed recipe does not resolve: %v", err)
			}
			if err := safety.Validate(composeRaw, res); err != nil {
				t.Fatalf("the proposed compose file breaks a safety rule: %v\n%s", err, composeRaw)
			}
			// The header comment names `ports:` in order to forbid it, so look for
			// the key at a service's own indentation rather than anywhere at all.
			if strings.Contains(string(composeRaw), "\n    ports:") {
				t.Error("the proposed compose file still publishes a port")
			}
			if !strings.Contains(files["recipe.yaml"], "TODO") {
				t.Error("a proposal with no TODO marker is claiming to be finished")
			}
		})
	}
}

func TestContainerPortTakesTheContainerSide(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{"8000:8000", 8000},
		{"127.0.0.1:8080:80", 80},
		{"3000", 3000},
		{"8080:80/tcp", 80},
		{map[string]any{"target": 5432, "published": 15432}, 5432},
		{"not-a-port", 0},
	}
	for _, tc := range cases {
		if got := containerPort(tc.in); got != tc.want {
			t.Errorf("containerPort(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestLiteralEnvRefusesToCarryTheOperatorsEnvironment(t *testing.T) {
	// An unset variable silently expanding to the empty string is how a volume mount
	// becomes "/". Nothing that references the operator's shell survives.
	for _, in := range []string{"${SECRET}", "$SECRET", "prefix-${A}-suffix"} {
		got, changed := literalEnv(in)
		if !changed || strings.Contains(got, "$") {
			t.Errorf("literalEnv(%q) = %q, %v; want a literal with no reference left", in, got, changed)
		}
	}
	if got, changed := literalEnv("Europe/Berlin"); changed || got != "Europe/Berlin" {
		t.Errorf("literalEnv left an ordinary value alone: got %q, %v", got, changed)
	}
}

func TestInputNameIsAlwaysASchemaName(t *testing.T) {
	cases := map[string]string{
		"/usr/src/paperless/media": "media",
		"/var/lib/data/":           "data",
		"/2fa":                     "data_2fa",
		"/My-Data":                 "my_data",
	}
	for in, want := range cases {
		if got := inputName(in); got != want {
			t.Errorf("inputName(%q) = %q, want %q", in, got, want)
		}
	}
}
