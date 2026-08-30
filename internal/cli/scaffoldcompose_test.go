package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The upstream Miniflux compose file, which is what the session 4 fresh-clone
// reviewer actually fed to `recipe init --compose`. Three of their findings came out
// of this one file, so it is the fixture.
const miniflux = `services:
  miniflux:
    image: ghcr.io/miniflux/miniflux:latest
    ports:
      - "80:8080"
    depends_on:
      db:
        condition: service_healthy
    environment:
      - DATABASE_URL=postgres://miniflux:secret@db/miniflux?sslmode=disable
      - RUN_MIGRATIONS=1
  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=miniflux
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=miniflux
    volumes:
      - miniflux-db:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "miniflux"]
volumes:
  miniflux-db:
`

func scaffoldMiniflux(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte(miniflux), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := detectCompose(path)
	if err != nil {
		t.Fatalf("detecting the compose file: %v", err)
	}
	files, err := scaffoldFromCompose("miniflux", d)
	if err != nil {
		t.Fatalf("scaffolding: %v", err)
	}
	return files["compose.yaml"]
}

// FC-03. The harness starts every service at once, so an application that treats a
// refused first database connection as fatal exits before the database is listening -
// and the ready probe then spends its whole budget failing to resolve a container
// that is no longer there. The scaffold used to drop both keys.
func TestScaffoldEmitsStartupOrdering(t *testing.T) {
	out := scaffoldMiniflux(t)

	if !strings.Contains(out, "healthcheck:") {
		t.Error("the database service has no healthcheck")
	}
	if !strings.Contains(out, "pg_isready") {
		t.Error("the healthcheck does not use pg_isready")
	}
	if !strings.Contains(out, "condition: service_healthy") {
		t.Error("the application does not wait for the database to be healthy")
	}
	if !strings.Contains(out, "depends_on:") {
		t.Error("the application has no depends_on")
	}
}

// FC-04. The scaffold rewrites the database service's own credentials to
// ${RESTORED_VAR_*} and used to leave the application's connection string frozen at
// the contributor's, so the two halves of the generated file disagreed about the
// password and the stack could not come up.
func TestScaffoldRepointsTheConnectionString(t *testing.T) {
	out := scaffoldMiniflux(t)

	if strings.Contains(out, "secret") {
		t.Error("the contributor's database password survived into the generated file")
	}
	want := "postgres://${RESTORED_VAR_db_user}:${RESTORED_VAR_db_password}@db/${RESTORED_VAR_db_name}?sslmode=disable"
	if !strings.Contains(out, want) {
		t.Errorf("DATABASE_URL was not repointed at the minted credentials.\nwant a line containing:\n  %s\ngot:\n%s", want, out)
	}
}

// MNT-10. Most real compose files say :latest, the safety schema rejects it, and the
// scaffold passed it straight through - so `recipe init --compose` wrote a recipe
// that failed the very next command the scaffold told the contributor to run.
func TestScaffoldDoesNotEmitLatest(t *testing.T) {
	out := scaffoldMiniflux(t)

	// The comment explaining the substitution mentions :latest, so look at the image
	// lines rather than at the whole file.
	for _, line := range strings.Split(out, "\n") {
		field := strings.TrimSpace(line)
		if !strings.HasPrefix(field, "image:") {
			continue
		}
		if image := strings.Fields(strings.TrimPrefix(field, "image:")); len(image) > 0 &&
			strings.HasSuffix(image[0], ":latest") {
			t.Errorf("the generated compose file still pins :latest, which recipe validate rejects: %s", field)
		}
	}
	if !strings.Contains(out, "your compose file said :latest") {
		t.Error("nothing tells the contributor why their tag was replaced")
	}
}

// A URL that does not point at the detected database service is left alone. The
// rewrite is narrow on purpose: it is guessing, and guessing wrong about somebody
// else's connection string is worse than not guessing.
func TestConnectionStringRewriteIsNarrow(t *testing.T) {
	db := &detectedDB{Kind: "postgres-dump", Service: "db", User: "miniflux", Password: "secret", Name: "miniflux"}

	cases := map[string]bool{
		"postgres://u:p@db/name":        true,  // the database service: rewritten
		"postgres://u:p@otherhost/name": false, // somebody else's database
		"https://example.com/webhook":   false, // not a database at all
		"postgres://db/name":            false, // no credentials to replace
		"not a url at all":              false,
		"redis://cache:6379":            false,
	}
	for in, wantChanged := range cases {
		_, changed := rewriteDBURL(in, db)
		if changed != wantChanged {
			t.Errorf("rewriteDBURL(%q) changed = %v, want %v", in, changed, wantChanged)
		}
	}

	// With no postgres database detected, nothing is ever rewritten.
	if _, changed := rewriteDBURL("postgres://u:p@db/name", nil); changed {
		t.Error("rewrote a connection string with no database detected")
	}
}
