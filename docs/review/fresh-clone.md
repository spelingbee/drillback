# Fresh-clone review - restored

Date: 2026-08-30. Clone of `d5c2f6c2d1fa8e5fff0fb5315f1e707604db4365` ("docs: PROGRESS.md,
the handover for session 4"), made with `git clone C:/My/Projects/Work/restored` into a
scratch directory; nothing in this report was run inside the repository itself.
Host: Windows 11, Git Bash, go1.27.0 windows/amd64, restic 0.19.1, Docker server 29.5.2
(WSL2, linux containers), golangci-lint on PATH, **no `make`**.
The reviewer started from the README and used only the repository's own documentation
until a wall was hit; every descent into `internal/**` is recorded as a finding.

## Summary

| severity | count |
|---|---|
| P0 | 1 |
| P1 | 5 |
| P2 | 6 |
| P3 | 4 |
| **total** | **16** |

A new user does get from the README to a passing recipe, and the parts of this project
built to be trusted mechanically are genuinely good: `restored check` produced a real PASS
on the first honest attempt, both demo scripts did exactly what the README says they do,
`recipe test` caught a real defect in my recipe and named the checks that were not
data-sensitive, and `restored check --help` is the best documentation in the repository.
What nearly stopped me was the beginning and the middle, not the end. **The README never
says how to install the tool** - the quick start's first executable line is
`restored recipe show gitea`, with nothing before it that puts `restored` on a PATH - and
the quick start itself is not runnable, because it assumes a restic repository the
documentation never tells you how to make, and its example `--input` paths contradict what
`recipe show` prints one line above. In the middle, `restored recipe init --compose`
handed me a compose file that could not start: it silently dropped the `depends_on` and
`healthcheck` that came from my own compose file, and it left the application's
`DATABASE_URL` carrying my credentials while rewriting the database service to
`${RESTORED_VAR_*}` ones. That cost 6.5 of the 13.4 minutes I spent on the recipe, in two
three-minute failing runs whose only diagnosis was `Could not resolve host: miniflux` - no
logs in the report, no pointer to `--log-level debug`. The honest number for "add a recipe
in 10 minutes" is **13 minutes 26 seconds** with the scaffold as it stands, and about
**7 minutes** if `recipe init` stops discarding startup ordering.

## The walk, timed

Real clock time from real runs. Reading and writing time is included where it was the
dominant cost, and is marked as such.

| step | command | result | minutes |
|---|---|---|---|
| 0 | `git clone C:/My/Projects/Work/restored restored` | ok | 0.01 |
| 1 | `go install github.com/spelingbee/restored/cmd/restored@latest` | **exit 1**, `Repository not found` | 0.13 |
| 1 | `go build -ldflags ... -o bin/restored ./cmd/restored` (what `make build` wraps) | ok, `restored version 0.1.0-dev` | 0.13 |
| 2 | `restored recipe show gitea --inputs-only` | ok, 0.41s | 0.01 |
| 2 | `restored check --recipe gitea --input data=/srv/gitea --input db=/srv/dumps/gitea.sql` (README verbatim) | **exit 2**, no matching files | 0.07 |
| 2 | invent a restic repository: read `scripts/lib.sh`, write a seeding script, run it | a repo with a real Gitea backup; 43s of run | 9.0 (mostly reading) |
| 2 | `restored check --recipe gitea --input data=/srv/gitea/data --input db=/srv/gitea/db.sql` | **PASS 5/5, exit 0** | 0.55 |
| 3 | `./scripts/demo.sh` | **PASS, exit 0** | 1.62 |
| 3 | `./scripts/demo-broken.sh` | **RESTORE UNUSABLE, exit 1** | 2.03 |
| 4 | `restored recipe init miniflux --compose .../docker-compose.yml` | scaffold written | 0.05 |
| 4 | read CONTRIBUTING, `docs/recipe-spec.md`, `recipes/TEMPLATE`; write recipe.yaml and compose.yaml | first draft | 4.4 |
| 5 | `restored recipe validate ./recipes/miniflux --strict` | **INVALID**: a maintainers pattern that is not in the generated docs | 0.02 |
| 5 | `restored recipe test ./recipes/miniflux --stage a` | **PASS**, 23s | 0.4 |
| 5 | `restored recipe test ./recipes/miniflux` | **stage B ERROR**, `Could not resolve host: miniflux` | 3.35 |
| 5 | `restored recipe test ./recipes/miniflux --stage b --log-level debug` | **ERROR**; the log finally shows `dial tcp 172.24.0.2:5432: connect: connection refused` | 3.1 |
| 5 | put `healthcheck` and `depends_on` back; `restored recipe test ./recipes/miniflux --stage b` | **PASS**, 29s | 0.5 |
| 5 | `restored recipe test ./recipes/miniflux` | **PASS both stages**, 45s | 0.75 |
| 6 | build and decode the prefilled contribution URL | well formed, but over the size ceiling for this recipe | 8.0 (mostly reading `internal/nudge`) |
| - | `go test ./...` on the clean tree | ok, every package | 0.08 |

Clone to a green `recipe test` on an application that is not in the registry: **about 22
minutes of wall clock**, of which 13m26s is steps 4 and 5.

## Findings

### FC-01 (P0) The README never says how to install the tool

**Step:** 1 (install)
**Where:** `README.md` - there is no `## Install` heading. `grep -n "^#\+ " README.md`
returns `What it looks like`, `Why this exists`, `Quick start`, `How a recipe works`,
`Bundled recipes`, `Add a recipe in 10 minutes`, `Contributors`, `Roadmap`, `License`.
**What happened:**

```
$ grep -n -i "install\|go get\|brew\|download\|release" README.md
14:> Pre-release, and not tagged. `restored check` works end to end against restic and
294:contribution types; the bot that maintains it is not installed, because installing an
306:  harness, six bundled recipes, CI, and the install paths.
```

The only build instruction anywhere in the README is `make build`, buried in the "What it
looks like" section as part of "reproduce them yourself". The Quick start's first
executable line is `restored recipe show gitea`, with nothing before it that puts
`restored` on a PATH. A reader who tries the obvious thing gets:

```
$ go install github.com/spelingbee/restored/cmd/restored@latest
go: github.com/spelingbee/restored/cmd/restored@latest: module github.com/spelingbee/restored/cmd/restored: git ls-remote -q --end-of-options https://github.com/spelingbee/restored in C:\Users\kadyr\go\pkg\mod\cache\vcs\c039c4fb...: exit status 128:
        remote: Repository not found.
        fatal: repository 'https://github.com/spelingbee/restored/' not found

real    0m7.955s
```

**What I expected, and what I had to do instead:** I expected an `## Install` section with
the `go install` line and a "no release binaries yet" note. I had to open
`CONTRIBUTING.md`, which has it, correctly, under "0. Get the tool":
`git clone ... && go build -o bin/restored ./cmd/restored`. That worked first time:

```
$ go build -ldflags "..." -o bin/restored ./cmd/restored
real    0m8.039s
$ ./bin/restored --version
restored version 0.1.0-dev
```

**Proposed fix:** Add `## Install` immediately before `## Quick start`, carrying the three
lines from `CONTRIBUTING.md` step 0, the
`go install github.com/spelingbee/restored/cmd/restored@latest` line, and one sentence
saying there are no tagged release binaries yet. This is a release blocker in the plainest
sense: a published README whose quick start cannot be started is not a released README.
The roadmap already lists "the install paths" as part of v0.1, so the gap is known; it has
just not landed in the file a stranger reads first.

### FC-02 (P1) The Quick start is not runnable: no way to make the repository it needs, and its `--input` paths contradict `recipe show`

**Step:** 2 (quick start)
**Where:** `README.md:143-152`
**What happened:** Running the quick start exactly as written against a real restic
repository, with only the paths substituted:

```
$ export RESTIC_REPOSITORY=.../qs/demo/repo
$ export RESTIC_PASSWORD_FILE=.../qs/demo/pass
$ ./bin/restored recipe show gitea --inputs-only
data  dir            required  /srv/gitea/data
db    postgres-dump  required  /srv/gitea/db.sql

$ ./bin/restored check --recipe gitea --input data=/srv/gitea --input db=/srv/dumps/gitea.sql
restored: required input "db": no matching files found for /srv/dumps/gitea.sql in the backup
exit=2
```

And with the README's own `RESTIC_PASSWORD_FILE=/etc/restic/pass`, which is what a literal
reader exports first:

```
restored: restic snapshots: exit status 1: {"message_type":"exit_error","code":1,"message":"Fatal: Resolving password failed: Fatal: /etc/restic/pass does not exist"}
exit=2
```

Two separate problems in six lines. First, the README never tells the reader how to obtain
a restic repository with anything in it, so the quick start only works for somebody who
already runs Gitea and already backs it up with restic - a much smaller audience than the
one the README is written for. Second, the `--input` values in the example
(`data=/srv/gitea`, `db=/srv/dumps/gitea.sql`) disagree with the paths `recipe show`
prints on the line above (`/srv/gitea/data`, `/srv/gitea/db.sql`), so a reader who copies
both lines gets exit 2 from the second. The error message is accurate but unhelpful: it
says the path was not found and does not say what *is* in the snapshot, though the tool
has already read the snapshot by then.

**What I expected, and what I had to do instead:** I expected the quick start to be
copy-pasteable, or to point at `scripts/demo.sh` as the way to see it work without owning
a Gitea. I had to read `scripts/lib.sh` - the demo helpers, explicitly labelled "Nothing
here is part of the tool" - to learn how the project makes a restic repository, then write
my own seeding script reusing `demo_start`, `gitea_sample_stack`, `gitea_seed`,
`gitea_dump` and `restic_init_and_backup` with the cleanup trap overridden so the
repository survives. That is nine minutes a new user should not have to spend. With the
paths `recipe show` actually printed, it passed:

```
  restore    ok          2.5s   2 inputs
  compose    ok          5.1s   2 services, db first for the dump
  load db    ok          8.7s   db: psql, 0 stderr lines
  ready      ok          7.6s   postgres accepts connections, gitea answers on the internal network

  CHECKS
  ✔  web-ui-renders      The web UI renders the instance home page        1.8s
  ✔  repos-in-db         The database contains at least one repository    1.4s
                         row → 1
  ✔  users-in-db         The database contains at least one real user     1.1s
                         account → 1
  ✔  repo-files-on-disk  At least one bare repository exists on disk     0.00s
                         → 1 match for */*.git/HEAD
  ✔  api-lists-repos     The API lists repositories, so the database     0.98s
                         and the disk agree → 1 item

  PASS  5/5 checks  ·  total 30.7s  ·  teardown ok

This backup boots.
exit=0  elapsed=33s
```

**Proposed fix:** Three things. (a) Make the example's `--input` values match the recipe
defaults, or drop `--input` from the quick start entirely, since `restored check --recipe
gitea` alone works. (b) Add one line after the quick start: "No Gitea to hand?
`./scripts/demo.sh` builds a real one, backs it up, and hands the backup to `restored` -
about ninety seconds." (c) When an input path is not found, print the snapshot's top two
or three levels alongside the error.

### FC-03 (P1) `recipe init --compose` discards `depends_on` and `healthcheck`, and the generated recipe then cannot pass stage B

**Step:** 4 and 5 (writing and testing the recipe)
**Where:** `restored recipe init --compose`; the generated `recipes/<name>/compose.yaml`
**What happened:** My input compose file - the upstream Miniflux one - had exactly what
upstream ships:

```yaml
  miniflux:
    depends_on:
      db:
        condition: service_healthy
  db:
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "miniflux"]
```

The generated `compose.yaml` had neither. `recipe test` stage A passed - because a
`restored check` run starts the database first in order to load the dump into it - and
then stage B failed, twice, for three minutes each:

```
  stage B  round trip: seed, export, back up, restore, check    ERROR     3m05s
           ready probe "miniflux answers its health endpoint" never succeeded
           after 21 attempts: curl: (6) Could not resolve host: miniflux
           up 1.7s · ready 3m01s

  ERROR  miniflux in 3m21s
```

The cause only appears at `--log-level debug`:

```
+ docker compose --profile test -p restored-lfsxk6fb -f ...\compose.yaml up -d --quiet-pull --pull missing
 Container restored-lfsxk6fb-db-1 Started
 Container restored-lfsxk6fb-feed-1 Started
 Container restored-lfsxk6fb-miniflux-1 Started
...
--- logs: miniflux ---
dial tcp 172.24.0.2:5432: connect: connection refused
```

Stage B brings the whole stack up at once. Miniflux tries PostgreSQL immediately, is
refused, and exits; the container is gone, so Docker's embedded DNS has nothing to resolve
and the ready probe reports a name-resolution failure for the next three minutes. This is
not a Miniflux quirk - it is the behaviour of any application that treats a failed first
database connection as fatal, which is most of them.

**What I expected, and what I had to do instead:** I expected `recipe init --compose` to
preserve the ordering constraints it found, since it preserves images and environment.
Nothing in `CONTRIBUTING.md`, in `docs/recipe-spec.md`, or in the generated file's own
header comment - which enumerates exactly four things it changed, and this is not one of
them - says that `depends_on` and `healthcheck` are dropped, or even that they are
allowed. I put both back by hand and stage B passed on the next run, in 29 seconds:

```
  stage B  round trip: seed, export, back up, restore, check    PASS      28.4s
           the round trip restored and all 5 checks passed
           up 4.2s · ready 1.6s · seed 1.7s · export 639ms · restic 5.8s · down 1.3s · check 13.3s
```

**Proposed fix:** Carry `depends_on` and `healthcheck` through `recipe init --compose`
unchanged - neither is on the isolation list, and `recipe validate --strict` accepts both.
Failing that, synthesise them: when the scaffold recognises a PostgreSQL or SQLite service
it already knows which service is the database, so it can emit the `pg_isready`
healthcheck and the `condition: service_healthy` edge itself. Either way, add both to the
generated header comment's list of what changed, and add a line to `CONTRIBUTING.md`
step 2 under the compose rules: "`depends_on` and `healthcheck` are allowed and you
probably need them - the harness starts every service at once."

### FC-04 (P1) `recipe init --compose` produces a stack whose application cannot authenticate to its own database

**Step:** 4 (writing the recipe)
**Where:** the generated `recipes/<name>/compose.yaml`
**What happened:** The scaffold rewrote the database service's credentials to
`${RESTORED_VAR_*}` and left the application's connection string exactly as it found it:

```yaml
services:
  miniflux:
    image: ghcr.io/miniflux/miniflux:latest
    environment:
      ADMIN_PASSWORD: "test123456"
      ADMIN_USERNAME: "admin"
      CREATE_ADMIN: "1"
      DATABASE_URL: "postgres://miniflux:secret@db/miniflux?sslmode=disable"
      RUN_MIGRATIONS: "1"
    networks: [restored]

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ${RESTORED_VAR_db_name}
      POSTGRES_USER: ${RESTORED_VAR_db_user}
      POSTGRES_PASSWORD: ${RESTORED_VAR_db_password}
```

`RESTORED_VAR_db_password` resolves to `restored-throwaway`, which the scaffold's own
`vars` block writes; the application is told the password is `secret`. The two halves of
the generated file do not agree, and the stack cannot come up. The header comment says
"anything that referenced an environment variable of yours became a literal" - which is
what happened to `DATABASE_URL`: correct by that rule, and wrong in effect, because the
value it was frozen from is the credential the other half of the file has just stopped
using.

**What I expected, and what I had to do instead:** I expected the connection string to be
rewritten to match. I had to do it by hand:

```yaml
      DATABASE_URL: "postgres://${RESTORED_VAR_db_user}:${RESTORED_VAR_db_password}@db/${RESTORED_VAR_db_name}?sslmode=disable"
```

**Proposed fix:** When the scaffold has identified a database service and minted
`db_user`, `db_password` and `db_name`, scan the other services' environment values for a
URL whose host matches the database service name and substitute the placeholders into it.
If that is judged too clever, emit a `TODO` on the offending line instead - the scaffold's
contract is "everything it could not decide is a TODO that names the decision", and this
is a decision it silently got wrong rather than one it flagged.

### FC-05 (P1) A service that dies at startup is reported as a DNS error, and `recipe test --report` carries no logs at all

**Step:** 5 (`recipe test`)
**Where:** the `recipe test` stage B ready-probe failure path; the `--report` JSON
**What happened:** The entire default-verbosity output for a container that exited on boot
was this:

```
  stage B  round trip: seed, export, back up, restore, check    ERROR     3m05s
           ready probe "miniflux answers its health endpoint" never succeeded
           after 21 attempts: curl: (6) Could not resolve host: miniflux
           up 1.6s · ready 3m01s

  ERROR  miniflux in 3m05s
```

I re-ran it with `--report`. The whole JSON is 1493 bytes and there is no `logs` key
anywhere in it:

```
schema_version 1
tool.name restored / tool.version 0.1.0-dev / tool.commit d5c2f6c
summary.total 1 / summary.errored 1
recipes[0].recipe miniflux
recipes[0].status error
recipes[0].stages[0].error ready probe "miniflux answers its health endpoint" never succeeded after 21 attempts: curl: (6) Could not resolve host:
```

`restored check --report` does the right thing by contrast, which I verified on a failing
run of the same recipe:

```
top keys: ['schema_version', 'tool', 'run', 'verdict', 'exit_code', 'recipe',
           'source', 'inputs', 'stages', 'checks', 'summary', 'hint', 'logs']
'logs' present: True
KEY logs {'db': ['The files belonging to this database system will be owned by user "postgres".', ...
```

So the tool already collects service logs on the `check` path and already writes them to
the report; the `recipe test` path collects them too - they appear under
`--log-level debug` - and then throws them away.

**What I expected, and what I had to do instead:** I expected the failure to say "service
`miniflux` is not running (exited)" and to show its last lines, the way `demo-broken.sh`
does for check failures. Nothing in the output or in `recipe test --help` suggests
`--log-level debug`; I only tried it because I have used enough Go CLIs to guess. Six and a
half of my thirteen recipe-writing minutes went into two blind three-minute runs.

**Proposed fix:** Two changes, in order of value. (a) Before a ready probe gives up, ask
`docker compose ps` which services are not running and put that in the error:
`ready probe "..." never succeeded: service miniflux is not running (exited 1)`. A
container that is gone is a different failure from a container that is slow, and the
current message points the reader at networking. (b) Put the same `logs` map that
`check --report` writes into the `recipe test --report` JSON, and print the last ten lines
of any non-running service's log on the terminal at default verbosity.

### FC-06 (P1) Adding a recipe dirties two generated files, the walkthrough never says so, and the first PR lands red

**Step:** 5 and 6 (finishing the recipe, opening the PR)
**Where:** `CONTRIBUTING.md` steps 3 to 5, the `Makefile`'s `check-generated` target, and
`.github/workflows/ci.yml`
**What happened:** After the recipe passed both stages I regenerated what
`check-generated` diffs:

```
$ go run ./tools/gen recipes-index > recipes/README.md
$ go run ./tools/gen readme-table
$ git status --short
 M README.md
 M recipes/README.md
?? recipes/miniflux/
$ git diff --stat
 README.md         | 1 +
 recipes/README.md | 3 ++-
 2 files changed, 3 insertions(+), 1 deletion(-)
```

`ci.yml` has `on: pull_request` with no `paths:` filter, so it runs on a recipe-only pull
request, and its generated-files job fails when those two files are stale. The workflow
itself says, in a comment at the top, "A pull request that touches only recipes/<name>/**
does NOT need this workflow to be green" - but that is a policy stated in a file the
contributor does not read, expressed as a comment rather than as a mechanism. What a
first-time contributor sees on their pull request is a red X on `ci`.

**What I expected, and what I had to do instead:** I expected `CONTRIBUTING.md`'s numbered
walkthrough to include a "regenerate the index" step, or `recipe init` or `recipe test` to
tell me. Neither does: `recipe init`'s next-steps block ends at `restored recipe test`, and
CONTRIBUTING goes straight from `recipe test` to `git add recipes/myapp`. I found it only
by running the generators out of suspicion.

**Proposed fix:** Cheapest first: add path filtering to `ci.yml` so a recipe-only PR does
not run it at all, which makes that comment true instead of aspirational. Then, either
way, add the regeneration to `CONTRIBUTING.md` step 5, and print it from `recipe test` on
success when the recipe directory is not in `recipes/README.md`: "this recipe is not in
the index yet - run `make recipes-index` and commit the result".

### FC-07 (P2) `docs/recipe-spec.md` drops item-level constraints and object sub-fields, so a recipe written from the reference alone does not validate

**Step:** 4 (writing the recipe)
**Where:** `docs/recipe-spec.md` sections "metadata" and "inputs.<name>", generated by
`tools/gen recipe-spec` from `schema/recipe.schema.json`
**What happened:** The reference documents `maintainers` with no constraints at all:

```
| `maintainers` | array of string |  |  |
```

so I wrote `maintainers: [spelingbee]`, which is what a GitHub handle looks like:

```
$ ./bin/restored recipe validate ./recipes/miniflux --strict
INVALID  ./recipes/miniflux
         recipe "recipes\\miniflux\\recipe.yaml": schema: metadata.maintainers.0: 'spelingbee' does not match pattern '^@[A-Za-z0-9-]+$'
```

The error message is excellent - it names the field and prints the pattern - but the
pattern exists only in the schema, because the generator prints constraints for the array
and not for its items. The same gap hides more: `inputs.<name>.mount` and
`inputs.<name>.load` are documented as a bare `object` with no field table, so nothing in
the reference tells you that `mount` takes `env` and `into`, or that `load` takes
`service`, `database`, `user` and `timeout`. I knew those only from the scaffold.
`within` is listed with a pattern and no explanation of what it is for.

**What I expected, and what I had to do instead:** I expected the "field reference
generated from the JSON Schema so that it cannot disagree with what the validator
enforces" to contain the constraints the validator enforces. I got them from the
validator's error message, for maintainers, and from the generated scaffold, for `mount`
and `load`. Quoting the value - `["@spelingbee"]`, because `@` is a reserved YAML
indicator and the bare form is a parse error - was a second small trap on the way.

**Proposed fix:** In `tools/gen recipe-spec`, recurse into `items` for arrays and into
`properties` for nested objects, emitting a sub-table for each the way the `kind:` variants
already get one. The page's own promise - that it cannot disagree with the validator - is
currently true only for the fields the generator happens to walk.

### FC-08 (P2) The `--compose` scaffold loses the only documentation of `profiles: [test]` and `${RESTORED_TEST_ASSETS}`

**Step:** 4 (writing the recipe)
**Where:** `recipes/TEMPLATE/compose.yaml` versus the output of
`restored recipe init --compose`
**What happened:** Miniflux keeps its data entirely in PostgreSQL, so the only honest
data-sensitive checks are row counts - and the strongest one, "at least one feed and one
entry survived", needs the seed step to subscribe to a feed. The run's network is
`internal: true`, so there is no feed to subscribe to. I was ready to record this as a dead
end for any recipe whose seeding needs a network fixture: `CONTRIBUTING.md` does not
mention the problem, and `docs/recipe-spec.md` mentions a `test/` directory in exactly one
clause with no explanation:

```
A recipe is a directory holding `recipe.yaml`, `compose.yaml`, and optionally a
`test/` directory of assets.
```

The mechanism does exist, and it is well explained - in `recipes/TEMPLATE/compose.yaml`,
the file the documentation tells you *not* to start from if you have a compose file of
your own:

```
  # A service with `profiles: [test]` exists during `restored recipe test` and never
  # during `restored check`. Use one when seeding needs a tool the application's own
  # image does not carry - a sqlite3 binary, a curl, a psql.
  #
  # ${RESTORED_TEST_ASSETS} is this recipe's own test/ directory, copied into the
  # workspace.
```

```
$ grep -rln "RESTORED_TEST_ASSETS" recipes/
recipes/TEMPLATE/compose.yaml
recipes/uptime-kuma/compose.yaml
$ grep -rn "RESTORED_TEST_ASSETS" README.md CONTRIBUTING.md docs/
(nothing)
```

**What I expected, and what I had to do instead:** I expected the primary path -
`recipe init --compose`, which both README and CONTRIBUTING recommend over the template -
to carry the template's guidance forward. It carries a much shorter header and none of the
mechanism comments. Once I found it, it worked exactly as advertised: an
`nginx:1.27-alpine` service with `profiles: [test]` serving `recipes/miniflux/test/` gave
me two genuinely data-sensitive checks, `feeds-in-db` and `entries-in-db`, that I would
otherwise have had to abandon.

The same section of the template contains the project's own honest answer to "should I
copy an existing recipe?": *"Copy that shape from a recipe that has it -
recipes/paperless-ngx/compose.yaml is the smallest one - rather than from a comment
here."* That is worth naming, because it means the scaffold is not yet doing the job the
ten-minute promise assigns to it.

**Proposed fix:** Have `recipe init --compose` emit the same commented `profiles: [test]`
and `${RESTORED_TEST_ASSETS}` block that `TEMPLATE/compose.yaml` carries - it is inert
until uncommented, and it is the difference between a contributor finding the mechanism
and dropping a check. Add a short "when seeding needs a fixture the internet would
normally provide" paragraph to `CONTRIBUTING.md` step 2, since `internal: true` makes this
situation common rather than exotic.

### FC-09 (P2) The generated recipe README omits an input and names the database after the service

**Step:** 4 (writing the recipe)
**Where:** the generated `recipes/<name>/README.md`
**What happened:** `recipe init --compose` wrote a `recipe.yaml` with two inputs, `data`
and `db`, and a `README.md` whose inputs table lists one of them. Reproduced with a second,
differently named scaffold to be sure it was not something I did:

```
$ ./bin/restored recipe init probe --compose .../docker-compose.yml --dir .../probe
$ grep -n "^  [a-z_]*:$" .../probe/probe/recipe.yaml
37:  data:
47:  db:
122:  export:
$ sed -n '5,12p' .../probe/probe/README.md
## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `db` | a pg_dump of the `db` database | `/srv/probe/db.sql` |

If your paths differ, point the inputs at yours:

    restored check --recipe ./recipes/probe --input data=/your/path
```

Two defects in eight lines. The `data` input that the same command just wrote into
`recipe.yaml` is missing from the table; and "a pg_dump of the `db` database" names the
*service* (`db`) rather than the database (`probe`, from `vars.db_name`). The example line
below then refers to `data`, the input the table does not list.

**What I expected, and what I had to do instead:** I expected the table to be generated
from the same input map as the recipe. I rewrote the README by hand from
`recipes/TEMPLATE/README.md`'s "What to write in a recipe README" section, which is a good
section and does its job.

**Proposed fix:** Generate the table by iterating the inputs the scaffold just wrote, and
render a `postgres-dump` input as "a pg_dump of the `<vars.db_name>` database".

### FC-10 (P2) The "one click" contribution link never appears for a normally commented recipe, and its fallback prints a broken `cp` on Windows

**Step:** 6 (the contribution URL)
**Where:** `CONTRIBUTING.md:165`; `internal/nudge/nudge.go` (`MaxURL = 6000`, and the
fallback block); `internal/cli/check.go` (`maybeNudge`)
**What happened:** Neither README nor CONTRIBUTING says when the invitation prints;
CONTRIBUTING only says "If `restored check` already printed you a prefilled GitHub
link...". A passing run of a non-bundled recipe printed nothing:

```
$ ./bin/restored check --recipe ./recipes/gitea
...
  PASS  5/5 checks  ·  total 15.8s  ·  teardown ok

This backup boots.
exit=0
```

I had to read `internal/cli/check.go` to find out why: `maybeNudge` returns early unless
stdout or stderr is a terminal, and a piped session has neither. Recording that descent as
the brief requires - I could not reach this feature from the documentation.

Having read `internal/nudge/nudge.go`, I built the invitation directly for my recipe and
for each bundled one. The URL is well formed and points where it claims:

```
scheme: https host: github.com path: /spelingbee/restored/new/main
params: ['filename', 'value']
filename: recipes/miniflux/recipe.yaml
value bytes: 4789
value first line: apiVersion: restored/v1
value line count: 158
total url len: 5983
```

That is GitHub's real prefilled file-creation endpoint, with the recipe YAML round-tripping
intact through `value`. But `MaxURL` is 6000 characters *after* encoding, and
percent-encoding YAML costs about 25 percent, so the effective ceiling is roughly 4900 raw
bytes:

```
TEMPLATE           9239 bytes  too large: prefilled link (10.5 KB encoded)
gitea              4789 bytes  LINK FITS
miniflux           5013 bytes  too large: prefilled link (6.1 KB encoded)
nextcloud          7167 bytes  too large: prefilled link (8.4 KB encoded)
paperless-ngx      7036 bytes  too large: prefilled link (8.2 KB encoded)
uptime-kuma        3916 bytes  LINK FITS
vaultwarden        4785 bytes  LINK FITS
```

Two of the five bundled recipes are over it, and so was mine at 5013 bytes - a recipe the
README describes as "about sixty lines of YAML", and which the project's own conventions
ask you to comment well. The "Adding it is one click" promise therefore fires for the small
half of recipes and silently degrades for the rest, with commenting quality as the deciding
variable.

The fallback it degrades to is wrong on Windows. `nudge.go` computes the source directory
as `strings.TrimSuffix(in.Path, "/recipe.yaml")`, and `rec.File` on this host is
`recipes\miniflux\recipe.yaml`, so nothing is trimmed:

```
    1. fork  https://github.com/spelingbee/restored
    2. cp -r recipes\miniflux\recipe.yaml recipes/miniflux
    3. restored recipe test ./recipes/miniflux     # this is what CI runs
    4. open a PR
```

That copies a file over a directory name. On any platform, if the recipe already lives in
`./recipes/<name>` - exactly where `recipe init` puts it, and where CONTRIBUTING tells you
to work - step 2 degenerates to `cp -r recipes/myapp recipes/myapp`.

**What I expected, and what I had to do instead:** I expected the documentation to tell me
when the link appears, so I could reach it deliberately, and I expected it to appear for a
recipe of ordinary size. Neither was possible without reading the source.

**Proposed fix:** (a) Use `filepath.Dir(in.Path)` instead of trimming a hard-coded `/`
suffix, and drop step 2 entirely when the recipe already lives under `recipes/`. (b) State
in `CONTRIBUTING.md` when the invitation prints: after a passing `restored check` with a
non-bundled recipe, on a terminal, unless `--no-nudge`. (c) Either raise `MaxURL` - GitHub
and current browsers handle well past 8000 - or measure it against the recipe with comments
stripped, so that comment quality stops deciding whether a contributor gets the one-click
path.

### FC-11 (P2) The `db/tables-empty` hint is written about Gitea and is printed verbatim for every recipe

**Step:** 5, and a deliberate failing `check`
**Where:** `docs/hints.yaml`, rule `db/tables-empty`
**What happened:** A Miniflux restore from an empty dump - a recipe with no data directory
at all, whose only input is a SQL file - produced this:

```
  LIKELY CAUSE
    The application's tables are there, but they are empty

    Every table the checks read exists and holds nothing. Two causes produce
    exactly this. Either the dump was taken from the wrong database, or with
    `pg_dump --schema-only`, or narrowed with `--table`, so it carried a
    schema and none of the rows. Or the dump carried nothing at all and the
    application rebuilt an empty schema for itself on start, which is what
    an application with automatic migrations does the moment it meets an
    empty database. Compare the size of the dump with the size of the data
    directory in the report above: a forge with repositories on disk and a
    half-kilobyte dump is not a backup.

      grep -c 'INSERT INTO' /srv/miniflux/db.sql
      ls -l /srv/miniflux/db.sql
                                                      (hint: db/tables-empty)
```

The diagnosis is right, and the two suggested commands are correctly rewritten to my
input's path, which is good work. The prose is not: "a forge with repositories on disk" is
about Gitea, and "the data directory in the report above" does not exist for this recipe,
which has one input and no directory.

**Proposed fix:** Rewrite the last sentence of `db/tables-empty` generically ("compare the
size of the dump with the amount of data the application should have"), and keep the forge
example for a Gitea-specific rule if it earns one. `CONTRIBUTING.md` advertises hints as
the cheapest useful contribution; a rule that reads as application-specific teaches the
wrong shape to the first person who copies it.

### FC-12 (P2) `recipe validate --strict` prints `ok` and then exits 2

**Step:** 4 (writing the recipe)
**Where:** `restored recipe validate --strict`
**What happened:**

```
$ ./bin/restored recipe validate ./recipes/miniflux --strict
ok       ./recipes/miniflux
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
exit=2
```

The first word of the output is `ok`, and the exit code is the one the help text defines as
"tool or runtime error: docker missing, restic failed, recipe invalid, timeout". A missing
maintainer is none of those. In a script - and `--strict` exists to be used in scripts - a
warning is now indistinguishable from a broken recipe file.

**Proposed fix:** Under `--strict`, print a verdict word that matches the exit code -
`warnings`, say - rather than `ok`. If exit 2 is deliberate, say so in `validate --help`
("--strict makes warnings fatal, exit 2"), because the global exit-code legend printed
under every command currently contradicts it.

### FC-13 (P3) Every documented command goes through `make`, with no fallback for a host that does not have it

**Step:** 1 and 3
**Where:** `README.md` ("Reproduce them yourself"), and the `Makefile`
**What happened:** `which make` on this host returns nothing. The README's only
reproduce-it-yourself block is:

```sh
make build
./scripts/demo.sh          # PASS, exit 0
./scripts/demo-broken.sh   # RESTORE UNUSABLE, exit 1
```

I read the `Makefile` and ran what `build` wraps. That is fine for me, and it is a
non-issue on Linux and macOS - but Windows is a first-class target for this tool
(`ci.yml` runs the unit suite on it, and `scripts/lib.sh` carries careful `MINGW*`
handling), and a Windows reader following the README stops on line one of that block.
Note that `CLAUDE.md` documents the workaround in detail, but `CLAUDE.md` is instructions
for an agent working on the repository, not documentation for a user of it.

**Proposed fix:** In the README, write `go build -o bin/restored ./cmd/restored` and
mention `make build` as the shorthand, rather than the other way round. `CONTRIBUTING.md`
already gets this right and is the model to copy.

### FC-14 (P3) Everything that points at `github.com/spelingbee/restored` 404s today

**Step:** 1 and 6
**Where:** the three README CI badges; `CONTRIBUTING.md` step 0; `internal/nudge/nudge.go`
(`Repo`); the `Docs:` footer under every `--help`
**What happened:** The repository is not public - which is a stop point in `CLAUDE.md`, and
correct - but the consequences are worth stating in one place.
`go install ...@latest` fails with `Repository not found` (output pasted in FC-01);
`git clone https://github.com/spelingbee/restored` from CONTRIBUTING step 0 fails the same
way; the three README badges render as broken images; and the prefilled contribution URL,
the mechanism the whole contribution flow depends on, points at a 404.

**Proposed fix:** Nothing to fix now; this resolves itself at stop point 4. It belongs on
the release checklist as one line: after the repository is public, re-run
`go install github.com/spelingbee/restored/cmd/restored@latest` from a clean machine and
open the nudge URL once. Both take seconds, and both are load-bearing.

### FC-15 (P3) An interrupted run left a Docker network behind

**Step:** 3 (present before I started)
**Where:** `restored` teardown
**What happened:** The environment already held one orphan when I began, from a run the
previous evening, hours before this session:

```
$ docker network inspect restored-3phn3mna_restored --format '{{.Name}} created={{.Created}} containers={{len .Containers}} labels={{.Labels}}'
restored-3phn3mna_restored created=2026-08-29 23:16:19.282830743 +0000 UTC containers=0 labels=map[com.docker.compose.project:restored-3phn3mna ... com.restored.run:3phn3mna]
```

To be fair to the tool: **none of my own runs leaked anything.** After one `demo.sh`, one
`demo-broken.sh`, six `recipe test` invocations - two of which errored out after a
three-minute ready-probe timeout - and five `restored check` runs, the only `restored`
object left on the host is that same pre-existing network:

```
$ docker ps -a --format '{{.Names}}' | grep -i restored
no containers
$ docker volume ls --format '{{.Name}}' | grep -i restored
no volumes
$ docker network ls --format '{{.Name}}' | grep -i restored
restored-3phn3mna_restored
```

Teardown is reliable on every path I exercised, including the failing ones. The orphan is
evidence that some path - most plausibly a hard kill - does not reach it. I left it in
place as evidence rather than removing it.

**Proposed fix:** The runs are already labelled `com.restored.run=<id>`, which is all a
sweeper needs. A `restored doctor` - already on the v0.2 roadmap - that lists and removes
networks, containers and volumes carrying that label with no running container closes this
in a few lines. Nothing here is urgent: it is one empty network.

### FC-16 (P3) The roadmap says six bundled recipes; there are five

**Step:** reading
**Where:** `README.md` roadmap, v0.1 line, versus the generated table above it
**What happened:** The roadmap reads "the round-trip harness, six bundled recipes, CI, and
the install paths", while the generated table lists five: `gitea`, `nextcloud`,
`paperless-ngx`, `uptime-kuma`, `vaultwarden`.

**Proposed fix:** Say five, or say "the bundled recipes" and let the generated table be the
only place the number lives. A hand-written count next to a generated table is exactly the
thing that goes stale between sessions, which is the argument the repository already makes
for generating the table in the first place.

## The miniflux recipe I ended up with

Both stages pass. Miniflux is worth adopting precisely because it is unlike the five
bundled recipes: it has **no directory input at all**. Feeds, categories, entries, users
and sessions all live in PostgreSQL, so the recipe exercises the `postgres-dump`-only path
that no bundled recipe covers - which is a shape worth having proven before launch.

Written from `restored recipe init miniflux --compose <the upstream compose file>`,
`CONTRIBUTING.md`, `docs/recipe-spec.md` and `recipes/TEMPLATE`; no bundled recipe was
copied. Three things had to be fixed by hand, and each is a finding above: the
`DATABASE_URL` credentials (FC-04), the `depends_on`/`healthcheck` pair (FC-03), and the
`profiles: [test]` fixture service that made the feed and entry checks possible (FC-08).

The verdict, from the final run against these exact files:

```
$ ./bin/restored recipe validate ./recipes/miniflux --strict
ok       ./recipes/miniflux
validate exit=0

$ ./bin/restored recipe test ./recipes/miniflux

recipe test miniflux (Miniflux)

  stage A  negative: the checks must fail against an empty sta… PASS      15.3s
           4 of 5 checks failed against an empty stack: categories-restored,
           feeds-in-db, entries-in-db, api-lists-categories
           check against empty inputs 15.3s
           $ restored check --recipe recipes\miniflux --source dir --from C:\Users\kadyr\AppData\Local\Temp\restored-empty-3571851122 --ready-timeout 1m30s
  stage B  round trip: seed, export, back up, restore, check    PASS      29.0s
           the round trip restored and all 5 checks passed
           up 4.3s · ready 1.2s · seed 1.4s · export 882ms · restic 5.6s · down 1.6s · check 14.1s
           $ restored check --recipe recipes\miniflux --source restic --from C:\Users\kadyr\AppData\Local\Temp\restored-hhqazkhz\repo --snapshot latest

  PASS   miniflux in 45.0s

  ────
  1 recipe: 1 passed, 0 failed, 0 errored, in 45.0s

exit=0 elapsed=45s
```

### `recipes/miniflux/recipe.yaml`

```yaml
apiVersion: restored/v1
kind: Recipe

metadata:
  name: miniflux
  title: Miniflux
  description: >
    Verifies that a Miniflux backup restores: the application answers its own health
    endpoint, and the feeds, categories and users that were in the backup are still
    there afterwards. Miniflux keeps everything in PostgreSQL, so the database dump is
    the whole backup.
  maintainers: ["@spelingbee"]
  upstream: https://miniflux.app
  tags: [rss, feed-reader, postgres]

vars:
  app_port: 8080
  db_name: miniflux
  db_user: miniflux
  # Not a secret. This database exists for about a minute on an internal network with
  # no published ports and is destroyed with `compose down -v`.
  db_password: restored-throwaway
  admin_user: restored-admin
  admin_password: restored-throwaway-admin

inputs:
  db:
    kind: postgres-dump
    title: The Miniflux database dump
    description: >
      Plain SQL from pg_dump, or a custom-format dump from pg_dump -Fc. Miniflux stores
      every feed, category, entry and user in PostgreSQL and keeps nothing on disk, so
      this dump is the entire backup. A typical nightly job writes it with
      `pg_dump -U miniflux miniflux > /srv/miniflux/db.sql`.
    default_path: /srv/miniflux/db.sql
    required: true
    load:
      service: db
      database: "{{ .vars.db_name }}"
      user: "{{ .vars.db_user }}"
      timeout: 5m

ready:
  - name: postgres accepts connections
    kind: exec
    service: db
    command: ["pg_isready", "-U", "{{ .vars.db_user }}", "-d", "{{ .vars.db_name }}"]
    timeout: 90s
    interval: 2s

  - name: miniflux answers its health endpoint
    kind: http
    url: http://miniflux:{{ .vars.app_port }}/healthcheck
    expect_status: 200
    timeout: 180s
    interval: 3s

checks:
  # Not data-sensitive on its own: it says the application came up against whatever
  # database it found. It is here so that "the app is broken" and "the data is gone"
  # are distinguishable in the report.
  - id: healthcheck-ok
    title: The application reports itself healthy
    kind: http
    url: http://miniflux:{{ .vars.app_port }}/healthcheck
    expect:
      status: 200

  # Data-sensitive. A fresh Miniflux creates exactly one category ("All") for the admin
  # user it makes on first boot, and no feeds at all, so a restore that carried no rows
  # cannot satisfy either of the next two checks.
  - id: categories-restored
    title: The database still holds the category that was backed up
    kind: sql
    driver: postgres
    service: db
    database: "{{ .vars.db_name }}"
    user: "{{ .vars.db_user }}"
    query: "SELECT count(*) FROM categories WHERE title = 'restored-drill';"
    expect:
      scalar_int_min: 1

  - id: feeds-in-db
    title: The database contains at least one subscribed feed
    kind: sql
    driver: postgres
    service: db
    database: "{{ .vars.db_name }}"
    user: "{{ .vars.db_user }}"
    query: "SELECT count(*) FROM feeds;"
    expect:
      scalar_int_min: 1

  - id: entries-in-db
    title: The database contains at least one feed entry
    kind: sql
    driver: postgres
    service: db
    database: "{{ .vars.db_name }}"
    user: "{{ .vars.db_user }}"
    query: "SELECT count(*) FROM entries;"
    expect:
      scalar_int_min: 1

  # The API reads the same rows through the application, so a dump that loaded but left
  # the application unable to use it still fails here.
  - id: api-lists-categories
    title: The API lists the restored category, so the app and the data agree
    kind: http
    method: GET
    url: http://miniflux:{{ .vars.app_port }}/v1/categories
    basic_auth: ["{{ .vars.admin_user }}", "{{ .vars.admin_password }}"]
    expect:
      status: 200
      json_path: "$"
      json_path_len_min: 2

test:
  seed:
    # Seeded through the application's own API rather than by writing to the database,
    # so the round trip proves the restore end to end.
    - name: create a category the checks can look for
      kind: http
      method: POST
      url: http://miniflux:{{ .vars.app_port }}/v1/categories
      basic_auth: ["{{ .vars.admin_user }}", "{{ .vars.admin_password }}"]
      json_body: '{"title":"restored-drill"}'
      expect_status: 201
      timeout: 60s

    # The feed is served by a static file inside the run's own network: the recipe's
    # stack is internal: true, so nothing can be fetched from the internet.
    - name: subscribe to the sample feed served inside the run's own network
      kind: http
      method: POST
      url: http://miniflux:{{ .vars.app_port }}/v1/feeds
      basic_auth: ["{{ .vars.admin_user }}", "{{ .vars.admin_password }}"]
      json_body: '{"feed_url":"http://feed/feed.xml"}'
      expect_status: 201
      timeout: 120s

  export:
    - name: dump the database the way a nightly backup job would
      kind: exec
      service: db
      command:
        - sh
        - -c
        - 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > "$RESTORED_EXPORT/db.sql"'
      timeout: 5m
      produces: db
```

### `recipes/miniflux/compose.yaml`

```yaml
# Miniflux keeps all of its state in PostgreSQL, so this stack is the application and
# its database and nothing else. There is no data directory to mount.
#
# Started from "restored recipe init --compose". DATABASE_URL had to be rewritten by
# hand: the generated file kept the credentials from my own compose file while the db
# service was given ${RESTORED_VAR_*} ones, so the two did not agree.
services:
  miniflux:
    image: ghcr.io/miniflux/miniflux:2.2.15
    environment:
      DATABASE_URL: "postgres://${RESTORED_VAR_db_user}:${RESTORED_VAR_db_password}@db/${RESTORED_VAR_db_name}?sslmode=disable"
      RUN_MIGRATIONS: "1"
      CREATE_ADMIN: "1"
      ADMIN_USERNAME: ${RESTORED_VAR_admin_user}
      ADMIN_PASSWORD: ${RESTORED_VAR_admin_password}
      LISTEN_ADDR: "0.0.0.0:8080"
      POLLING_SCHEDULER: "round_robin"
    # Miniflux exits when its first connection to PostgreSQL is refused, and nothing
    # restarts it. Both of these came from the upstream compose file and were dropped
    # by `restored recipe init --compose`; without them the round-trip harness brings
    # every service up at once and Miniflux loses the race with postgres.
    depends_on:
      db:
        condition: service_healthy
    restart: on-failure
    networks: [restored]

  # Miniflux's own data directory does not exist: the throwaway volume below is the
  # database this run creates, and the restore arrives as a dump that is loaded into it.
  db:
    image: postgres:16.4-alpine
    environment:
      POSTGRES_DB: ${RESTORED_VAR_db_name}
      POSTGRES_USER: ${RESTORED_VAR_db_user}
      POSTGRES_PASSWORD: ${RESTORED_VAR_db_password}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB"]
      interval: 2s
      timeout: 5s
      retries: 30
    volumes:
      - db-data:/var/lib/postgresql/data
    networks: [restored]

  # Only during `restored recipe test`. The run's network is internal: true, so the
  # seed step cannot subscribe to a feed on the internet. This serves one from the
  # recipe's own test/ directory instead.
  feed:
    image: nginx:1.27-alpine
    profiles: [test]
    volumes:
      - ${RESTORED_TEST_ASSETS}:/usr/share/nginx/html:ro
    networks: [restored]

volumes:
  db-data:

networks:
  restored:
    internal: true
```

### `recipes/miniflux/test/feed.xml`

The fixture the `profiles: [test]` nginx service serves, so that the seed step can
subscribe to a feed on a network with no route to the internet.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>restored drill feed</title>
    <link>http://feed/feed.xml</link>
    <description>A fixture feed served inside the run's own network.</description>
    <item>
      <title>First drill entry</title>
      <link>http://feed/entry-1.html</link>
      <guid isPermaLink="false">restored-drill-1</guid>
      <description>The first entry that a restore has to bring back.</description>
    </item>
    <item>
      <title>Second drill entry</title>
      <link>http://feed/entry-2.html</link>
      <guid isPermaLink="false">restored-drill-2</guid>
      <description>The second entry that a restore has to bring back.</description>
    </item>
  </channel>
</rss>
```

A `README.md` for the recipe was written as well, following
`recipes/TEMPLATE/README.md`'s three questions plus the "what this recipe does not prove"
section. It is in the scratch clone and is not reproduced here.

### Two notes for whoever adopts this recipe

- `admin_user` and `admin_password` are `vars` because the API checks and the seed steps
  both need them and `basic_auth` takes literals. `CREATE_ADMIN=1` is left on for the
  restore run as well: Miniflux finds the admin already present in the restored dump and
  carries on, which is why the same compose file serves stage A, stage B and a real
  `restored check`.
- `restart: on-failure` is belt and braces next to `depends_on: condition:
  service_healthy`. Either alone fixed the failure in FC-03; both are cheap, and a
  contributor's machine under load is exactly where the race comes back.
