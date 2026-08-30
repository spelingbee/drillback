package cli

import (
	"fmt"
	"regexp"
	"strings"
)

var recipeName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// scaffold renders the three files `drillback recipe init` writes. The result validates
// against the schema and the safety rules, so a contributor's first command after
// init succeeds and their second one is about their application rather than about
// YAML.
func scaffold(name, db, image string, dirs []string) (map[string]string, error) {
	if !recipeName.MatchString(name) {
		return nil, fmt.Errorf("recipe name %q: expected 3 to 40 lower-case letters, digits and hyphens", name)
	}
	if len(dirs) == 0 {
		dirs = []string{"data"}
	}
	if image == "" {
		image = "example/" + name + ":1.0.0"
	}

	var recipeYAML, composeServices strings.Builder

	fmt.Fprintf(&recipeYAML, `apiVersion: drillback/v1
kind: Recipe

metadata:
  name: %s
  title: %s
  description: >
    Verifies that a %s backup restores: the application serves its own pages and the
    data that was backed up is still there. Replace this with what your checks prove.
  # Put your GitHub handle here. drillback does not guess it.
  maintainers: []
  tags: []

vars:
  app_port: 8080

inputs:
`, name, title(name), title(name))

	for _, d := range dirs {
		fmt.Fprintf(&recipeYAML, `  %s:
    kind: dir
    title: The %s directory
    description: >
      What this directory holds on a default docker compose install, and where it
      usually lives on the host.
    default_path: /srv/%s/%s
    required: true
    mount:
      env: DRILLBACK_INPUT_%s
      into: app:/%s
`, d, d, name, d, d, d)
	}

	switch db {
	case "postgres-dump":
		recipeYAML.WriteString(`  db:
    kind: postgres-dump
    title: The application database dump
    description: >
      Plain SQL from pg_dump, or a custom-format dump from pg_dump -Fc. The format is
      detected from the file's magic bytes, not from its extension.
    default_path: /srv/` + name + `/db.sql
    required: true
    load:
      service: db
      database: "{{ .vars.db_name }}"
      user: "{{ .vars.db_user }}"
      timeout: 5m
`)
	case "sqlite":
		recipeYAML.WriteString(`  db:
    kind: sqlite
    title: The application SQLite database
    description: >
      The .db file itself. It is declared separately so the SQL checks have something
      to point at, and so a zero-byte database is reported as a restore failure
      instead of as a mysteriously empty application.
    default_path: /srv/` + name + `/` + dirs[0] + `/app.db
    within: ` + dirs[0] + `
    required: true
    load:
      integrity_check: true
`)
	}

	recipeYAML.WriteString(`
ready:
  - name: the application answers on the internal network
    kind: http
    url: http://app:{{ .vars.app_port }}/
    expect_status: 200
    timeout: 180s
    interval: 3s

checks:
  # NEITHER of these checks is data-sensitive: both of them pass against an empty
  # application. That makes this recipe worthless as it stands, and writing a check
  # that FAILS on an empty stack is the first and most important thing you do.
  #
  # A data-sensitive check reads something only a real restore can produce: a row
  # count, a file the user created, an API listing that is empty on a fresh install.
  - id: app-answers
    title: The application serves its home page
    kind: http
    url: http://app:{{ .vars.app_port }}/
    expect:
      status: 200

  - id: data-dir-present
    title: The data directory exists in the restored tree
    kind: file
    service: app
    path: /` + dirs[0] + `
    expect:
      exists: true
      is_dir: true
`)

	if db != "none" {
		recipeYAML.WriteString(`
  # Replace the table name and make this the check that proves the restore.
  # - id: rows-present
  #   title: The database contains at least one real row
  #   kind: sql
  #   driver: ` + sqlDriver(db) + `
`)
	}

	fmt.Fprintf(&composeServices, `services:
  app:
    image: %s
    environment:
      # Whatever the application needs in order to start without its usual host.
      APP_PORT: "${DRILLBACK_VAR_app_port}"
    volumes:
`, image)
	for _, d := range dirs {
		fmt.Fprintf(&composeServices, "      - ${DRILLBACK_INPUT_%s}:/%s\n", d, d)
	}
	composeServices.WriteString("    networks: [drillback]\n")

	if db == "postgres-dump" {
		composeServices.WriteString(`
  db:
    image: postgres:16.4-alpine
    environment:
      POSTGRES_DB: ${DRILLBACK_VAR_db_name}
      POSTGRES_USER: ${DRILLBACK_VAR_db_user}
      POSTGRES_PASSWORD: ${DRILLBACK_VAR_db_password}
    volumes:
      - db-data:/var/lib/postgresql/data
    networks: [drillback]

volumes:
  db-data:
`)
	}
	composeServices.WriteString(`
networks:
  drillback:
    internal: true
`)

	compose := `# Rendered by drillback before ` + "`docker compose up`" + `. The ${DRILLBACK_INPUT_*} values
# are absolute paths inside the run workspace, and ${DRILLBACK_VAR_*} come from vars:
# plus any --set overrides.
#
# Do not add ports:, privileged:, network_mode:, pid:, ipc:, or a bind mount to any
# path outside the workspace. ` + "`drillback recipe validate`" + ` rejects all of them.
` + composeServices.String()

	readme := fmt.Sprintf(`# %s

What this recipe assumes about your backup.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
%s
If your paths differ, point the inputs at yours:

    drillback check --recipe ./recipes/%s --input %s=/your/path

## What the checks prove

Replace this section with the answer to one question: **which of these checks would
fail if the backup were empty?** If the answer is "none", the recipe is not finished.
`, title(name), inputRows(dirs, db, name), name, dirs[0])

	out := map[string]string{
		"recipe.yaml":  recipeYAML.String(),
		"compose.yaml": compose,
		"README.md":    readme,
	}
	if db == "postgres-dump" {
		out["recipe.yaml"] = strings.Replace(out["recipe.yaml"],
			"vars:\n  app_port: 8080\n",
			"vars:\n  app_port: 8080\n  db_name: "+name+"\n  db_user: "+name+"\n"+
				"  # Not a secret. This database exists for about ninety seconds on an internal\n"+
				"  # network with no published ports, and is destroyed with `compose down -v`.\n"+
				"  db_password: drillback-throwaway\n", 1)
	}
	return out, nil
}

func sqlDriver(db string) string {
	if db == "sqlite" {
		return "sqlite"
	}
	return "postgres"
}

func inputRows(dirs []string, db, name string) string {
	var b strings.Builder
	for _, d := range dirs {
		fmt.Fprintf(&b, "| `%s` | the %s directory | `/srv/%s/%s` |\n", d, d, name, d)
	}
	switch db {
	case "postgres-dump":
		fmt.Fprintf(&b, "| `db` | a pg_dump of the database | `/srv/%s/db.sql` |\n", name)
	case "sqlite":
		fmt.Fprintf(&b, "| `db` | the SQLite database | `/srv/%s/%s/app.db` |\n", name, dirs[0])
	}
	return b.String()
}

func title(name string) string {
	parts := strings.Split(name, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
