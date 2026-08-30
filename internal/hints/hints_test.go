package hints

import (
	"strings"
	"testing"
)

func TestBuiltinCatalogLoads(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatalf("the built-in catalog must load: %v", err)
	}
	if len(c.Rules) == 0 {
		t.Fatal("the built-in catalog is empty")
	}
	for _, r := range c.Rules {
		if r.re == nil {
			t.Errorf("rule %q has no compiled pattern", r.ID)
		}
	}
}

// fixtures pairs every rule in docs/hints.yaml with a string it must match and one it
// must not. A rule with no fixture is a rule nobody has ever seen fire.
var fixtures = map[string]struct{ match, nearMiss string }{
	"postgres/relation-missing": {
		`ERROR:  relation "repository" does not exist`,
		`ERROR:  column "repository" does not exist`,
	},
	"postgres/empty-dump": {
		`pg_restore: error: did not find magic string in file header`,
		`pg_restore: restoring data for table "repository"`,
	},
	"postgres/role-missing": {
		`ERROR:  role "gitea" does not exist`,
		`ERROR:  database "gitea" does not exist`,
	},
	"postgres/version-mismatch": {
		`pg_restore: error: unsupported version (1.16) in file header`,
		`pg_restore: processing item 1 ENCODING ENCODING`,
	},
	"postgres/auth-failed": {
		`FATAL:  password authentication failed for user "gitea"`,
		`LOG:  database system is ready to accept connections`,
	},
	"sqlite/not-a-database": {
		`Error: file is not a database`,
		`Error: no such column: id`,
	},
	"sqlite/wal-missing": {
		`Error: database disk image is malformed`,
		`Error: table monitor already exists`,
	},
	"sqlite/empty-schema": {
		`Error: no such table: monitor`,
		`Error: no such column: monitor`,
	},
	"compose/image-pull-failed": {
		`Error response from daemon: manifest unknown`,
		`Error response from daemon: container already exists`,
	},
	"compose/port-conflict": {
		`Error starting userland proxy: bind: address already in use`,
		`Error starting container: no such image`,
	},
	"permissions/eacces": {
		`open /data/gitea/conf/app.ini: permission denied`,
		`open /data/gitea/conf/app.ini: no such file or directory`,
	},
	"restore/empty-input": {
		`restored input "data" is empty`,
		`restored input "data" is 1.8 GiB`,
	},
	"restore/path-not-in-snapshot": {
		`no matching files found for "/srv/gitea/data"`,
		`restored 14203 files from snapshot 4a7f1c2e`,
	},
	"app/still-in-setup": {
		`<h1>Installation</h1> redirecting to /install?lang=en`,
		`<h1>Dashboard</h1>`,
	},
	"compose/service-exited-at-boot": {
		`curl: (6) Could not resolve host: miniflux`,
		`dial tcp 172.24.0.2:5432: connect: connection refused`,
	},
	"docker/daemon-unreachable": {
		`Cannot connect to the Docker daemon at unix:///var/run/docker.sock`,
		`docker: 'compose' is not a docker command`,
	},
	"workspace/no-space": {
		`write /var/tmp/restored-k7m2q9xf/inputs/data: no space left on device`,
		`write /var/tmp/restored-k7m2q9xf/inputs/data: input/output error`,
	},
	"db/tables-empty": {
		"expected scalar_int_min: 1, got 0",
		"expected scalar_int_min: 1, got 7",
	},
}

func TestEveryRuleHasAFixture(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range c.Rules {
		f, ok := fixtures[r.ID]
		if !ok {
			t.Errorf("rule %q has no fixture: add one to hints_test.go", r.ID)
			continue
		}
		if !r.re.MatchString(f.match) {
			t.Errorf("rule %q does not match its own fixture %q", r.ID, f.match)
		}
		if r.re.MatchString(f.nearMiss) {
			t.Errorf("rule %q matches its near miss %q, so it is too broad", r.ID, f.nearMiss)
		}
	}
	for id := range fixtures {
		found := false
		for _, r := range c.Rules {
			if r.ID == id {
				found = true
			}
		}
		if !found {
			t.Errorf("fixture %q has no rule: the catalog and the test have drifted", id)
		}
	}
}

// TestRuleOrdering asserts that the specific rules still win. Adding a broad rule
// above a specific one silently changes the diagnosis every user sees, and this is
// the test that says so.
func TestRuleOrdering(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ subject, want, driver string }{
		{`ERROR:  relation "repository" does not exist`, "postgres/relation-missing", "postgres"},
		{`ERROR:  role "gitea" does not exist`, "postgres/role-missing", "postgres"},
		{`Error: file is not a database`, "sqlite/not-a-database", "sqlite"},
		{`Error: no such table: monitor`, "sqlite/empty-schema", "sqlite"},
		{"expected scalar_int_min: 1, got 0", "db/tables-empty", "postgres"},
	}
	for _, tc := range cases {
		rule, where, ok := c.Match([]Subject{{Where: "checks[0]", Text: tc.subject, Driver: tc.driver}})
		if !ok {
			t.Errorf("nothing matched %q", tc.subject)
			continue
		}
		if rule.ID != tc.want {
			t.Errorf("%q matched %q, want %q", tc.subject, rule.ID, tc.want)
		}
		if where != "checks[0]" {
			t.Errorf("matched_on = %q", where)
		}
	}
}

// TestDriverScoping is the reason `when:` exists: a Postgres rule must not fire on a
// SQLite failure that happens to contain the same words.
func TestDriverScoping(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	subject := `relation "monitor" does not exist`
	if _, _, ok := c.Match([]Subject{{Text: subject, Driver: "sqlite"}}); ok {
		t.Error("a postgres-scoped rule fired on a sqlite failure")
	}
	if _, _, ok := c.Match([]Subject{{Text: subject, Driver: "postgres"}}); !ok {
		t.Error("a postgres-scoped rule did not fire on a postgres failure")
	}
}

func TestAtMostOneHint(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	// Two subjects, both matching. The first one in order wins, and only one hint
	// comes back: a list of five possible causes is a list of five things to ignore.
	rule, where, ok := c.Match([]Subject{
		{Where: "checks[0].observed.error", Text: `ERROR:  relation "repository" does not exist`, Driver: "postgres"},
		{Where: "logs.db", Text: `Cannot connect to the Docker daemon`},
	})
	if !ok {
		t.Fatal("nothing matched")
	}
	if rule.ID != "postgres/relation-missing" || where != "checks[0].observed.error" {
		t.Errorf("matched %q at %q, want the first subject to win", rule.ID, where)
	}
}

func TestRenderCommands(t *testing.T) {
	c, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	var rule *Rule
	for i := range c.Rules {
		if c.Rules[i].ID == "postgres/relation-missing" {
			rule = &c.Rules[i]
		}
	}
	if rule == nil {
		t.Fatal("rule not found")
	}
	got := rule.RenderCommands(CommandContext{
		Inputs:     map[string]string{"db": "/srv/gitea/db.sql"},
		SnapshotID: "4a7f1c2e",
	})
	if len(got) == 0 {
		t.Fatal("no commands rendered")
	}
	for _, cmd := range got {
		if strings.Contains(cmd, "{{") {
			t.Errorf("command %q still holds a template", cmd)
		}
		if !strings.Contains(cmd, "/srv/gitea/db.sql") {
			t.Errorf("command %q does not name the user's own path", cmd)
		}
	}
}

func TestParseRejectsABrokenCatalog(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{"a pattern that does not compile",
			"version: 1\nrules:\n  - id: a/b\n    match: '('\n    title: t\n    text: x\n", "does not compile"},
		{"a rule with no id",
			"version: 1\nrules:\n  - match: x\n    title: t\n    text: x\n", "no id"},
		{"a duplicate id",
			"version: 1\nrules:\n  - id: a/b\n    match: x\n    title: t\n    text: y\n" +
				"  - id: a/b\n    match: z\n    title: t\n    text: y\n", "twice"},
		{"an unknown key",
			"version: 1\nrules:\n  - id: a/b\n    match: x\n    title: t\n    text: y\n    severity: high\n", "severity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.doc))
			if err == nil {
				t.Fatal("this catalog must be rejected")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
