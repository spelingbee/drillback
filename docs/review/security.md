# Security review - restored

Date: 2026-08-30. Commit: `d5c2f6c2d1fa8e5fff0fb5315f1e707604db4365` (`main`, clean tree).
Ran: `go build`, `go vet ./...` (clean), `go test ./...` (all green), `golangci-lint run`
(0 issues), `./bin/restored recipe validate --strict` and `recipe show` against 17
adversarial recipes written into a scratch directory, one `docker compose config` (parse
only, no daemon objects created), and a standalone Go program to demonstrate the
`filepath.Glob` traversal primitive.
Did NOT run: anything that starts a container - no `recipe test`, no `go test -tags
integration`, no `scripts/demo*.sh`, no `restored check`. Another reviewer owns Docker
this session. Findings about container runtime behaviour are reasoned from the code plus
`docker compose config` output and are labelled as such.
Also not run: `TestSanitiseNeutralisesEscapingSymlinks` skips on this host (Windows will
not create symlinks for an unprivileged user), so the symlink sanitiser was reviewed by
reading, not by execution.

## Summary

| Severity | Count |
|---|---|
| P0 | 2 |
| P1 | 3 |
| P2 | 4 |
| P3 | 4 |

The isolation model is well designed and, where it is enforced, it is enforced properly:
the argv discipline is real (nothing reaches a shell), input paths are checked for `..`
in two independent places, symlink neutralisation exists and is deliberate, secrets are
kept out of `docker run` argument lists by passing environment names rather than values,
and the report struct deliberately marks response bodies and stdout/stderr as
non-serialised. The problem is not the design; it is that the enforcement has two holes
that both defeat it completely, and both are reachable from a recipe that
`restored recipe validate --strict` calls `ok`.

The first is a YAML injection: a recipe variable is substituted into `compose.yaml`
*after* the safety schema has run and is never re-validated, so a newline in a `vars:`
value writes `privileged: true` into the file that `docker compose up` reads. The second
is that `schema/compose-safety.schema.json` constrains `services` and `networks` but
leaves the top-level `volumes:` block entirely unvalidated, so a named volume with
`driver_opts: {type: none, device: /, o: bind}` is a bind mount of the host root that the
schema's own volume rule never looks at. Either one converts a contributed recipe into
root on the machine running the drill, and into root on the CI runner that tests fork
pull requests.

Below those, the schema is a deny-list where it should be an allow-list (a service body
takes arbitrary extra keys, so `volumes_from`, `extra_hosts`, `uts`, `sysctls` and
`group_add` all pass), SPEC.md section 9.3 promises two secret-handling controls that do
not exist in the code, and `recipes.yml` interpolates a fork-controlled string into a
shell `run:` block. Nothing was found in the argv/command-injection class, and nothing
was found in the restored-tree path-traversal class beyond a read-only enumeration
oracle.

## Findings

### SEC-01 (P0) A recipe variable injects arbitrary YAML into the compose file that runs

**Where:** `internal/recipe/safety/interpolate.go:17-52` (`Interpolate`),
`internal/runner/runner.go:198-211`, `internal/harness/stageb.go:114-137`

**What:** `safety.ValidateSchema` runs against `compose.yaml` **as written**, with
`${RESTORED_*}` placeholders intact (this is deliberate, ADR-039). `Interpolate` then
substitutes the values and escapes exactly one character - `$` - before the result is
written to the workspace and handed to `docker compose up`. A `vars:` value containing a
newline therefore adds keys to the compose document, and nothing re-validates the
interpolated file: `CheckResolvedMounts` (the only post-interpolation check) parses
`services.*.volumes` and nothing else, and `LabelCompose` round-trips the YAML unchanged
apart from labels.

`vars` is `{"type": ["string","number","boolean"]}` in `schema/recipe.schema.json:29-33`
with no pattern, and the bundled Gitea recipe already models the vulnerable shape
(`POSTGRES_PASSWORD: ${RESTORED_VAR_db_password}`, unquoted, `recipes/gitea/compose.yaml:11`).

**Reproduction:**

`recipe.yaml`:

```yaml
vars:
  port: "8080\n    privileged: true\n    network_mode: host\n    pid: host"
```

`compose.yaml`:

```yaml
services:
  app:
    image: nginx:1.27-alpine
    environment:
      APP_PORT: ${RESTORED_VAR_port}
    volumes: ["${RESTORED_INPUT_data}:/app/data"]
    networks: [restored]
networks:
  restored:
    internal: true
```

```text
$ ./bin/restored recipe validate .../p15-yamlinject --strict
ok       C:/.../scratchpad/secrev/p15-yamlinject
exit=0

$ ./bin/restored recipe show .../p15-yamlinject --compose
# --- rendered compose.yaml ---
services:
  app:
    image: nginx:1.27-alpine
    environment:
      APP_PORT: 8080
    privileged: true
    network_mode: host
    pid: host
    volumes: ["<workspace>/inputs/data:/app/data"]
    networks: [restored]
networks:
  restored:
    internal: true
```

That rendered document is byte-for-byte what `runner.Run` writes to
`ws.ComposeFile()` and runs (`runner.go:198-220`); `recipe show --compose` calls the same
`safety.Interpolate`.

**Impact:** A contributed recipe that passes `recipe validate --strict` runs a privileged
container on the host network with the host PID namespace. That is root on the machine
running the drill, and root on the GitHub-hosted runner for every fork pull request that
`recipes.yml` tests. It falsifies SPEC.md section 9.3's first bullet
("All of these are schema-level hard failures ... enforced before anything is started")
and is in scope per SECURITY.md ("Escape from the run's isolation ... without
`restored recipe validate` refusing it").

Note for scoping: the *volume* variant of this injection is caught -
`CheckResolvedMounts` rejects an injected `/:/host`. `privileged`, `network_mode`, `pid`,
`ipc`, `cap_add`, `devices` and every other forbidden key are not.

**Proposed fix:** Two independent changes, both cheap:

1. Re-run `safety.ValidateSchema` (and `checkForbiddenKeys`) on the **interpolated**
   document, immediately after `Interpolate` and before `LabelCompose`, in both
   `runner.Run` and `harness.stageB`. The schema cannot recognise a `${RESTORED_*}` bind
   mount post-interpolation, so keep the existing pre-interpolation pass as well and run
   both. This is the change that closes the class, not just this instance.
2. Constrain `vars` values in `schema/recipe.schema.json` to a single-line pattern
   (e.g. `"pattern": "^[^\\n\\r]*$"` on the string branch), and make `Interpolate` reject
   a substituted value containing `\n`, `\r` or `#` with a named error.

---

### SEC-02 (P0) A named volume with `driver_opts` bind-mounts any host path, including `/` and the Docker socket

**Where:** `schema/compose-safety.schema.json:1-84` (no constraint on the top-level
`volumes` key; root `properties` lists only `services` and `networks` and the root has no
`additionalProperties: false`), `internal/recipe/safety/interpolate.go:157-163`
(`isHostPath`)

**What:** The volume rule at `compose-safety.schema.json:42-62` only inspects the *left
side of a service's volume string*. `hostroot:/host` matches the "named volume" branch
`[a-z][a-z0-9_-]*` and is accepted. The top-level `volumes:` block that defines
`hostroot` is never validated at all, so the `local` driver's documented
`type=none,o=bind,device=<path>` options turn that "named volume" into a bind mount of an
arbitrary host path. `CheckResolvedMounts` then skips it, because `isHostPath("hostroot")`
is false.

**Reproduction:**

```yaml
services:
  app:
    image: nginx:1.27-alpine
    volumes:
      - "${RESTORED_INPUT_data}:/app/data"
      - "hostroot:/host"
    networks: [restored]
volumes:
  hostroot:
    driver: local
    driver_opts:
      type: none
      device: /
      o: bind
networks:
  restored:
    internal: true
```

```text
$ ./bin/restored recipe validate .../p04-driveropts --strict
ok       C:/.../scratchpad/secrev/p04-driveropts
exit=0

$ ./bin/restored recipe validate .../p06-docksock --strict     # device: /var/run
ok       C:/.../scratchpad/secrev/p06-docksock
exit=0
```

The same file, normalised by compose itself (parse only, no containers started, no volume
created):

```text
$ RESTORED_INPUT_data=/tmp/ws/inputs/data docker compose -f compose.yaml config
...
volumes:
  hostroot:
    name: p04-driveropts_hostroot
    driver: local
    driver_opts:
      device: /
      o: bind
      type: none
```

The long-syntax route is equally accepted (`type: volume, source: hostroot`, probe
`p10-longvolopts`, `ok`, exit 0), and so is `external: true` with an arbitrary `name:`
(probe `p14-externalvol`, `ok`, exit 0) - which attaches a volume belonging to one of the
user's other containers.

Contrast with the direct route, which is correctly refused:

```text
$ ./bin/restored recipe validate .../p03-absbind --strict
INVALID  .../p03-absbind
         compose.yaml: services.app.volumes.1: '/:/host' does not match pattern
         '^(\$\{RESTORED_(INPUT|TEST_ASSETS|EXPORT)[A-Za-z0-9_]*\}|[a-z][a-z0-9_-]*):/[^:]+...'
exit=2
```

**Impact:** Full read/write access to the host filesystem from inside a recipe container,
or `-v /var/run:/var/run` which hands the container the Docker socket. The latter also
falsifies SPEC.md section 9.3's explicit mitigation "restored never mounts the Docker
socket **into** a container". Same blast radius and same reachability as SEC-01: a
stranger's recipe, `ok` under `--strict`, executed by CI on every fork PR.

I did not start a container to observe the mount, so this is proven up to and including
"compose accepts and preserves the driver options"; the runtime behaviour of the `local`
driver with `o=bind` is documented Docker behaviour, not something I measured here.

**Proposed fix:** Add a top-level `volumes` schema that forbids what makes a named volume
not a named volume, and set `additionalProperties: false` at the document root so a new
top-level key is a review decision rather than a silent pass:

```json
"volumes": {
  "type": "object",
  "additionalProperties": {
    "type": ["object", "null"],
    "additionalProperties": false,
    "properties": { "labels": true, "name": { "type": "string" } },
    "not": { "anyOf": [
      { "required": ["driver_opts"] }, { "required": ["driver"] },
      { "required": ["external"] } ] }
  }
}
```

Mirror it in `forbiddenService`-style Go so the error message names the key, the way
`checkForbiddenKeys` already does for services.

---

### SEC-03 (P1) A restic repository URL carrying a password is written verbatim into the report

**Where:** `internal/runner/runner.go:652-660` (`repositoryLabel`),
`internal/runner/runner.go:570`, `internal/report/tty.go:57`,
`internal/source/restic/restic.go:51-53`

**What:** `repositoryLabel` returns `--from` unchanged, or falls back to
`os.Getenv("RESTIC_REPOSITORY")`, and the result becomes `report.source.repository` - a
serialised field (`internal/source/source.go:31`). It is printed by the TTY report and
written by `--json` and `--report <file>`. restic's REST backend takes credentials in the
repository string itself (`rest:https://user:password@host:8000/`), which is documented
upstream and is a common configuration.

The code comment asserts the opposite of what the code does:

```go
// repositoryLabel is what the report shows for the repository. A repository string can
// carry a user name but never a password, and restored never reads the environment
// variables that do.
```

SPEC.md section 9.3 makes a stronger and equally untrue claim: "The JSON report contains
the repository *URL* but no credentials, and the URL is scrubbed of any `user:password@`
userinfo." There is no scrubbing anywhere in the tree:

```text
$ grep -rn -i 'scrub\|userinfo\|redact' --include='*.go' .
(no output)
```

The same string is echoed on stderr under `--log-level debug` by
`restic.Options.run`, which logs `+ restic --repo <value> ...` before executing.

**Impact:** A backup repository password lands in a JSON file users are explicitly
encouraged to attach to bug reports (`.github/ISSUE_TEMPLATE/*` asks for the report), in
CI artifacts, and on the terminal. SECURITY.md names this exact case as in scope:
"Secret disclosure. RESTIC_PASSWORD ... appearing in a report, a log, a process argument
list, or a JSON document."

**Proposed fix:** Implement the scrub SPEC.md already promises. Parse the part after the
backend prefix as a URL and, when `u.User` is set, replace it with the user name plus
`:***`, falling back to dropping the whole userinfo if parsing fails. Apply it in one
place - a `restic.SafeRepository(string) string` - and use it from `repositoryLabel` and
from `Options.run`'s debug line. Add a unit test with
`rest:https://u:p@h/r`, `s3:https://k:s@h/b`, `sftp:u@h:/p` and a plain local path.

---

### SEC-04 (P1) A compose service body accepts arbitrary keys, so the isolation deny-list is incomplete

**Where:** `schema/compose-safety.schema.json:11-67` (service body has no
`additionalProperties: false`), `internal/recipe/safety/safety.go:265-278`
(`forbiddenService`)

**What:** Both the schema and the Go rule enumerate keys to *reject*. Every compose key
not on that twelve-entry list is accepted. Probed and confirmed accepted (`exit=0`,
`--strict`):

```text
$ ./bin/restored recipe validate .../p07-volumesfrom --strict   # volumes_from: ["container:some-host-container"]
ok       .../p07-volumesfrom
$ ./bin/restored recipe validate .../p08-extrahosts --strict    # extra_hosts: ["reachme:host-gateway"], uts: host
ok       .../p08-extrahosts
$ ./bin/restored recipe validate .../p12-secopt --strict        # security_opt: ["label:disable"], sysctls, group_add
ok       .../p12-secopt
```

Of these, `volumes_from: ["container:<name>"]` is the concrete one: it attaches the
volumes of an existing container on the host to a recipe container. SPEC.md section 9.2
lists "The user's other running containers" as an asset worth protecting; nothing stops a
recipe reaching them. `extra_hosts` with `host-gateway` writes the host's gateway address
into the container's `/etc/hosts`; whether traffic then reaches the host depends on the
daemon's iptables rules for `internal: true` networks, which I did not test - treat that
one as an undeclared control gap rather than a demonstrated exploit. `uts: host`,
`security_opt: label:disable`, `sysctls` and `group_add` are lesser but are all decisions
the recipe should not be making.

**Impact:** The deny-list will keep losing to the compose specification, which grows. Each
new key is a silent, unreviewed grant. Today the exploitable one is `volumes_from`.

**Proposed fix:** Invert it. Put `additionalProperties: false` on the service body and
enumerate the keys a recipe legitimately needs (`image`, `networks`, `volumes`,
`environment`, `env_file`, `command`, `entrypoint`, `user`, `working_dir`, `depends_on`,
`healthcheck`, `profiles`, `labels`, `restart`, `tmpfs`, `stop_grace_period`, `cap_add`,
`security_opt`, `read_only`, `init`, `shm_size`, `mem_limit`, `pids_limit`). Keep
`checkForbiddenKeys` for the message quality on the common mistakes, and make the
"unknown key" error say which key and point at CONTRIBUTING.md. This also removes the
`forbiddenService`/schema duplication that the comment at `safety.go:263` already worries
about.

---

### SEC-05 (P1) GitHub Actions script injection from a fork-controlled directory name

**Where:** `.github/workflows/recipes.yml:125`, `:135`, `:146`, `:156` (and the same
shape at `:183`)

**What:** The `test` job's matrix is built from file paths in the pull request diff:

```sh
echo "$files" | grep -E '^recipes/[^/]+/' | cut -d/ -f2 | grep -v '^TEMPLATE$' | sort -u
```

and each value is then interpolated by GitHub into a shell `run:` block:

```yaml
- name: validate
  run: ./bin/restored recipe validate ./recipes/${{ matrix.recipe }} --strict
```

A directory name is contributor-controlled and git permits `;`, `$(`, backticks and
spaces in path components. GitHub expression interpolation happens before the shell sees
the script, so the value is not a shell argument - it is script text.

**Reproduction (selection pipeline reproduced locally; the workflow was not dispatched):**

```text
$ printf '%s\n' 'recipes/probe;id;x/compose.yaml' 'recipes/normal/recipe.yaml' \
  | grep -E '^recipes/[^/]+/' | cut -d/ -f2 | grep -v '^TEMPLATE$' | sort -u
normal
probe;id;x

# the run: step then becomes
./bin/restored recipe validate ./recipes/probe;id;x --strict
```

`recipe-health.yml` has the same interpolation but sources its matrix from `ls` on the
default branch, so it is not reachable from a PR.

**Impact:** Arbitrary command execution in the `recipes` job for a fork pull request. The
job is deliberately low-value - `pull_request` (not `pull_request_target`), read-only
`GITHUB_TOKEN`, no secrets, ephemeral runner - and it already runs contributor-chosen
container images, which is why this is P1 and not P0. It still breaks the reasoning the
workflow states about itself, it can poison the branch-scoped `actions/setup-go` cache,
and a public repository with a textbook Actions injection in it is a bad first
impression at launch.

**Proposed fix:** Never interpolate into `run:`. Pass through the environment and quote:

```yaml
- name: validate
  env:
    RECIPE: ${{ matrix.recipe }}
  run: ./bin/restored recipe validate "./recipes/$RECIPE" --strict
```

and, in the `pick` step, filter the selection to the same identifier pattern the recipe
schema already enforces: `grep -E '^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$'`. That filter is
worth having on its own - a directory whose name is not a legal `metadata.name` is not a
recipe. `${{ matrix.recipe }}` inside `with:` (the artifact name at `:146`) is safe and
can stay.

---

### SEC-06 (P2) `expect.glob` escapes the workspace and turns the report into a host filesystem oracle

**Where:** `internal/check/run.go:279-288`, `schema/recipe.schema.json:218`
(`"glob": {"type": "string"}` - no `not` guard)

**What:** Every other path field in the schema carries an explicit `..` guard:
`$defs/absolutePath` has `"not": {"pattern": "(^|/)\\.\\.(/|$)"}` and so does the `file`
check's `path` (`recipe.schema.json:269`). `expect.glob` has neither, and it is joined
straight onto the resolved workspace path:

```go
matches, globErr := filepath.Glob(filepath.Join(host, filepath.FromSlash(c.Expect.Glob)))
```

`filepath.Join` cleans the `..` away instead of refusing it. The match count is returned
as `observed.count`, which is a serialised report field
(`internal/check/expect.go:24`), and `glob_min_count` turns it into a pass/fail signal.

**Reproduction:**

```text
$ ./bin/restored recipe validate .../p16-tmplpath --strict   # glob: "../../../../*", glob_min_count: 1
ok       .../p16-tmplpath
exit=0
```

The traversal primitive itself, demonstrated with the same two calls the code makes:

```text
$ go run .   # host = C:/tmp/restored-abc123/inputs/data
glob "*.db"                                         -> C:\tmp\restored-abc123\inputs\data\*.db  matches=0 err=<nil>
glob "../../../../*"                                -> C:\*  matches=26 err=<nil>
glob "../../../../Windows/System32/drivers/etc/*"   -> C:\Windows\System32\drivers\etc\*  matches=6 err=<nil>
```

**Impact:** Read-only enumeration of paths outside the run workspace, with the answer
reported. It leaks file names and existence, not contents, so it is a P2 and not a P0 -
but SECURITY.md's in-scope list says "Anything that lets a recipe ... read ... a path
outside `<tmp>/restored-<runid>`", and this does exactly that.

**Proposed fix:** Add `"not": {"pattern": "(^|/)\\.\\.(/|$)"}` to `glob` in the schema for
consistency with `path`, and - because the schema validates the recipe *before*
templating, so a `{{ .vars.x }}` could still smuggle `..` past it - enforce containment at
use: after the `filepath.Join`, reject the result unless `Executor` can confirm it is
still under `best.HostPath`. The `check` package has no workspace handle today, so the
cheapest version is a `strings.HasPrefix(filepath.Clean(joined), filepath.Clean(host))`
guard in `Executor.File` returning `Observation{Error: ...}`.

---

### SEC-07 (P2) `recipe show` does not disclose the images a recipe will pull

**Where:** `internal/cli/recipe.go:223-239` (`shownRecipe` / `showDocument`)

**What:** "Recipes pull arbitrary images" is an accepted risk (SPEC.md section 9.3,
SECURITY.md "Out of scope"), and the accepted-risk argument depends on the user being
able to see what they are agreeing to before running it. `recipe show` prints metadata,
vars, resolved inputs, ready probes and checks. It prints no images at all.

**Reproduction:**

```text
$ ./bin/restored recipe show gitea | grep -c -i image
0

$ ./bin/restored recipe show gitea --inputs-only
data  dir            required  /srv/gitea/data
db    postgres-dump  required  /srv/gitea/db.sql
```

`recipes/gitea/compose.yaml` pulls `postgres:16.4-alpine` and `gitea/gitea:1.22.6`;
neither appears. `--compose` prints the whole rendered file, which is the raw material,
not a disclosure. Two further images are never disclosed anywhere: the check helper
(`curlimages/curl:8.16.0`, `internal/runner/runner.go:32`, overridable by
`RESTORED_HELPER_IMAGE`) and the harness's restic image
(`restic/restic:0.19.1`, `internal/harness/harness.go:39`, overridable by
`RESTORED_RESTIC_IMAGE`).

**Impact:** The one control the project offers against a hostile image - "read the recipe
first" - has no supported way to be exercised short of reading a second file by hand.
This is the disclosure the threat model leans on.

**Proposed fix:** Add an `images:` section to `showDocument`, parsed with the `Compose`
type `safety` already has (`safety.Parse` -> `ServiceNames()` -> `Services[n].Image`),
listing service, image, and whether it is digest-pinned. Include the helper image and, for
`recipe test`, the restic image, both marked as restored's own. Add
`--images-only` alongside the existing `--inputs-only`, and mention it in the README next
to "running a recipe from a stranger is running a container from a stranger".

---

### SEC-08 (P2) The JSON report embeds 200 lines of every service's container log

**Where:** `internal/runner/runner.go:508-527` (`collectLogs`), `:225`, `:261`, `:361`;
`internal/report/report.go:43` (`Logs map[string][]string \`json:"logs,omitempty"\``)

**What:** On any non-PASS verdict, `restored` collects `docker compose logs --tail 200`
for every service and serialises it into the report. Those logs are produced by the
application that has just been started **on top of the user's real restored data**. A
Nextcloud or Paperless log line names files and users; a Postgres log line can contain
statement text; an application that logs its own configuration on startup logs whatever
the compose `environment:` block gave it.

This is deliberate and useful - it is what makes a failure diagnosable - but it is not
labelled, and the report is the artifact the issue templates ask people to attach.
SECURITY.md's in-scope list explicitly includes "the contents of a restored backup
appearing in a report ... or a JSON document."

The related fields are handled correctly, which is worth saying: `Observation.Body`,
`Stdout` and `Stderr` are `json:"-"` (`internal/check/expect.go:29-32`) and the TTY report
never renders any of them. `Observation.Value` *is* serialised, so one scalar per `sql`
check - whatever the recipe's `SELECT` returned - is also in the report.

**Impact:** Restored backup content in a file users are prompted to attach to public
issues and that CI uploads as an artifact (`recipes.yml:142-151`, retention 14 days;
`recipe-health.yml:78-87`, 30 days). No exploit is needed; the normal path does it.

**Proposed fix:** Do not change the default behaviour - the logs are the point - but make
it visible and controllable. (a) Say it in `restored check --help` and in the `--report`
flag description: "the report includes the last 200 log lines of every service, which may
contain data from the backup." (b) Add `--no-logs` for anyone who intends to share the
report. (c) Print a one-line notice on the TTY when `--json` or `--report` is used and
`rep.Logs` is non-empty. (d) Document the same in SPEC.md section 5.2 so the JSON schema's
stability contract carries the warning.

---

### SEC-09 (P2) A `sql` check's `file:` is unconstrained and opens arbitrary host paths

**Where:** `schema/recipe.schema.json:255` (`"file": {"type": "string"}`),
`internal/check/run.go:192`, `internal/sqlite/sqlite.go:21-28`

**What:** For `driver: sqlite`, the recipe supplies `file:` and it goes straight to
`os.Stat` and then to `sql.Open("sqlite", dsn(file))`. Nothing requires it to be inside
the workspace. The intended form is `"{{ .inputs.db.path }}"` (see
`recipes/uptime-kuma/recipe.yaml`), but `file: /home/user/.local/share/keyrings/x.db` is
equally valid to the schema, and the resulting error text
(`opening x.db: no such file` vs `file is not a database`) is returned as
`observed.error`, a serialised report field.

The DSN is also built by string concatenation -
`url.URL{Scheme: "file", Opaque: file}.String() + "?" + q.Encode()`
(`sqlite.go:63-68`) - and `Opaque` is emitted unescaped, so a `?` in the path would inject
query parameters ahead of the `mode=ro` that is appended. That variant needs a host file
whose name contains `?` to exist, so it is not practically reachable; it is worth fixing
as brittleness, not as a live bug.

**Impact:** Existence probing of arbitrary host paths, reported. Read-only, no content
disclosure (the file has to be a valid SQLite database to yield rows, and `mode=ro`
prevents writes, including through `ATTACH`). Same class as SEC-06 and the same
containment invariant broken.

**Proposed fix:** Give `check.Executor` the workspace root and require a resolved
`sqlite` `file:` to be inside it, returning a named error otherwise
("check %q: file %s is outside the run workspace; use {{ .inputs.<name>.path }}"). Build
the DSN with `url.URL{Scheme: "file", Path: ...}` plus `RawQuery`, not with `Opaque` and
string concatenation.

---

### SEC-10 (P3) Release-work requirements: no `install.sh`, no signatures or SBOM, no image, no `docs/docker.md`

**Where:** `.goreleaser.yaml:35-47`, absence of `install.sh`, `Dockerfile`,
`docs/docker.md`; SPEC.md sections 12.2, 12.3, 12.5

**What:** None of the release artifacts exist yet, which is correct - tagging is a stop
point. Recording the security requirements on them now, as findings against the release
work, so they are not discovered at tag time:

- `.goreleaser.yaml` has `checksum: {name_template: checksums.txt}` and nothing else. SPEC
  12.2 requires "`checksums.txt` plus a cosign keyless signature of it, and SBOMs via
  syft" - there is no `signs:` block and no `sboms:` block. Both must exist before the
  first tag, because `install.sh` (SPEC 12.5 step 5) is specified to verify that
  signature and cannot be written against a signature that is not produced.
- `before: hooks: [go mod tidy]` mutates `go.mod`/`go.sum` during the release build, so
  the released binary is not built from the tree CI tested. Drop the hook and let CI fail
  on an untidy module instead. (Related: every requirement in `go.mod` is marked
  `// indirect`, including `cobra`, `yaml.v3`, `jsonschema/v6` and `modernc.org/sqlite`,
  which are direct - `go mod tidy` at release time would rewrite exactly this.)
- The ghcr image (SPEC 12.3) does not exist, and its documented invocation mounts
  `/var/run/docker.sock` into it. Requirements when it is written: the socket mount must
  never be baked into the image (no `VOLUME`, no entrypoint that assumes it), the image
  must contain a client only and no daemon, it must not run as root, `docs/docker.md`
  must lead with the socket-mount warning rather than bury it, and the README's docker
  snippet must not be copy-pasteable without the reader passing the warning. The line
  already in SPEC 12.3 (`# The socket mount is root-equivalent access to the host's
  Docker. Read section 9.`) is the right tone - keep it in the shipped docs, not only in
  the spec.
- `install.sh` does not exist. SPEC 12.5 steps 3-5 are the security-relevant ones and are
  correctly specified (download checksums *and* signature, exit non-zero on mismatch
  *before* extracting, never silently skip cosign). Requirement to add: the version
  resolved from `RESTORED_VERSION` must be validated against `^v[0-9]+\.[0-9]+\.[0-9]+`
  before it is interpolated into a URL.

**Impact:** None today - nothing is published. All of it blocks the first tag.

**Proposed fix:** As above. Fold the four bullets into the SPEC 12.6 release checklist so
they are checked at the stop point rather than remembered.

---

### SEC-11 (P3) The `vars` secret warning specified in SPEC.md section 9.3 is not implemented

**Where:** `internal/recipe/safety/safety.go:239-260` (`Warnings`), SPEC.md section 9.3

**What:** SPEC.md states as a mitigation: "`recipe validate --strict` warns on any var
whose name matches `(?i)(secret|token|apikey|api_key|private)` and whose value has high
entropy." `Warnings` checks three things - empty `metadata.maintainers`, fewer than two
checks, and an untagged image - and nothing about vars.

**Reproduction:** the three warning branches are the whole function; there is no third
argument to inspect vars with. `safety.Warnings(r *recipe.Recipe, composeRaw []byte)`.

**Impact:** Recipe `vars` are printed by `recipe show` and land in the report (SPEC 9.3
says so explicitly), so a contributor who puts a real credential in one gets no warning
before it is committed to a public repository. This is the control that was supposed to
catch that, and a maintainer reviewing a PR believes it ran.

**Proposed fix:** Implement it as specified, in `Warnings`, so `--strict` fails on it -
name match plus a cheap entropy floor (length >= 20 and >= 3 character classes) to keep
`db_password: restored-throwaway` from tripping it. Add the bundled recipes as the
negative test.

---

### SEC-12 (P3) Templates in `default_path` are silently not expanded, and the `..` guard runs pre-render

**Where:** `internal/recipe/resolve.go:110-121` and `:186-205` (`resolveOne` reads
`in.DefaultPath` before `res.renderRecipe()` at `:161`)

**What:** `Resolve` builds every `ResolvedInput.BackupPath` from the **unrendered**
`default_path`, then renders the recipe copy afterwards. A templated `default_path`
therefore keeps its braces:

```text
$ ./bin/restored recipe show .../p16-tmplpath --inputs-only
data  dir  required  /srv/probe/{{ .vars.esc }}/data
```

That path is what reaches `restic restore --include` and `dirsource.Locate`, so the
recipe simply restores nothing. It is a correctness bug today. It is a security-relevant
one tomorrow, because the moment someone "fixes" the ordering, the `not: ..` guard on
`$defs/absolutePath` - which is applied to the file as written - becomes bypassable by
putting `../..` in a `vars` value. `validBackupPath` (`resolve.go:207-220`) would still
catch it, which is the reason to keep that function rather than rely on the schema.

**Impact:** No live exploit. A trap for the next change in this area.

**Proposed fix:** Either render the inputs block before resolving and re-run
`validBackupPath` on the rendered value (which is already the correct order for
`mount.into`, rendered at `:167`), or reject a `default_path` containing `{{` in the
schema and say so. Whichever is chosen, add a test that asserts the chosen behaviour, so
this is a decision rather than an accident.

---

### SEC-13 (P3) Defence-in-depth: an unreachable `..` check, and argv passed to `docker compose exec` without `--`

**Where:** `internal/workspace/sanitise.go:80-84`; `internal/compose/compose.go:147-158`

**What:** Two hardening notes. Neither is an exploitable issue and both are labelled as
defence-in-depth.

1. `Sanitise` refuses "a path component equal to `..`", but the paths it inspects come
   from `filepath.WalkDir`, which builds them with `filepath.Join` and therefore cleans
   them, and no filesystem permits a directory entry literally named `..`. The check can
   never fire. The real protection against `..` in a restored tree is elsewhere
   (`validBackupPath`, and restic's own `--target` semantics), which SPEC.md section 9.3
   also says. The risk is that the comment at `sanitise.go:66` reads as a guarantee that
   the code does not provide, and someone relies on it. The genuinely load-bearing half
   of `Sanitise` - the symlink neutralisation - is correct: `WalkDir` does not descend
   into symlinked directories, so `filepath.Dir(p)` is always a real parent and the
   lexical `Join`+`Contains` resolution cannot be defeated by an intermediate link.
   I could not execute the test that covers it on this host (see the header).

2. `Client.Exec` appends the recipe's `command` argv directly after the service name with
   no `--` separator. `docker compose exec` sets non-interspersed flag parsing, so this is
   fine today; it depends on upstream CLI behaviour rather than on anything restored
   controls. `RunHelper` and `RunContainer` have the same shape after the image name.

**Impact:** None measured. Item 1 is a documentation-vs-code mismatch in a
security-critical file; item 2 is a dependency on someone else's parser.

**Proposed fix:** For (1), either delete the `..` loop and amend the comment to say what
actually protects the tree, or move the check somewhere it can fire (over the raw
`--include` results). For (2), insert `"--"` before `o.Argv` in `Exec`, `RunHelper` and
`RunContainer` - three lines, and it stops mattering what any future CLI version does
with a leading dash.

---

## Verified safe

Every item here was attacked and held. Commands and their real output.

**Compose isolation rules that are genuinely enforced.** All of these were written as
recipes and run through `./bin/restored recipe validate <dir> --strict`:

```text
p01 privileged: true
  INVALID  compose.yaml: service "app" uses `privileged`, which restored does not allow:
           a privileged container is not isolated from the host                        exit=2
p02 network_mode: host
  INVALID  compose.yaml: service "app" uses `network_mode`, ...                        exit=2
p05 x-danger: &danger {privileged: true, pid: host} + `<<: *danger` in the service
  INVALID  compose.yaml: service "app" uses `pid`, ...                                 exit=2
p03 volumes: ["/:/host"]
  INVALID  services.app.volumes.1: '/:/host' does not match pattern '^(\$\{RESTORED_...  exit=2
p09 volumes: [{type: bind, source: /, target: /host}]
  INVALID  services.app.volumes.1.type: value must be one of 'volume', 'tmpfs'         exit=2
p11 cap_add: [SYS_ADMIN]
  INVALID  services.app.cap_add.0: value must be one of 'CHOWN', 'DAC_OVERRIDE',
           'FOWNER', 'SETGID', 'SETUID'                                                exit=2
p13 networks.restored.external: true
  INVALID  networks.restored.external: value must be false                             exit=2
```

The YAML **merge key** attack (p05) is worth calling out: `gopkg.in/yaml.v3` resolves
`<<: *anchor` when decoding into `map[string]any`, so `checkForbiddenKeys` sees the merged
keys and rejects them. That was the most likely-looking bypass and it does not work.
`ports`, `ipc`, `userns_mode`, `devices`, `device_cgroup_rules`, `cgroup_parent`,
`build`, `extends`, `external_links` and top-level `include` are on the same list and are
rejected by the same code path (`safety.go:265-305`).

**Input path traversal.** Both the schema guard and the Go guard fire, independently:

```text
$ ./bin/restored recipe validate .../p17-dotdot        # default_path: "/srv/probe/../../../etc/data"
INVALID  ... schema: inputs.data.default_path: 'not' failed                             exit=2

$ ./bin/restored recipe show <r> --input 'data=/srv/../../../etc' --inputs-only
restored: input "data": path "/srv/../../../etc" contains ".."                          exit=2

$ ./bin/restored recipe show <r> --input 'data=../../../etc' --inputs-only
restored: input "data": path "../../../etc" is not absolute; paths inside a backup
are absolute POSIX paths                                                                exit=2
```

`validBackupPath` (`resolve.go:207`) runs on the recipe default *and* on the `--input`
override, so the CLI flag is covered by the same rule as the recipe field.

**No command injection was found.** I traced every recipe-controlled value that reaches an
exec argument. `internal/compose` and `internal/source/restic` are indeed the only
packages that call `os/exec` (`grep -rn 'os/exec' --include='*.go'` also returns
`internal/compose/env.go`, which only runs `docker version` / `restic version` with fixed
arguments). No `sh -c` is constructed anywhere; no command string is built by
concatenation. Recipe `command:` is a JSON/YAML array typed `{"type":"array","items":
{"type":"string"}}` and is passed as argv. A value beginning with `-` cannot be smuggled
into a flag position: every recipe-controlled string that precedes a positional argument
(`-u <user>`, `-U <user>`, `-d <database>`, `-c <query>`, `--include <path>`,
`--repo <repo>`, `--tag <tag>`) is consumed as that flag's value. `restic restore` takes
`snap.ID` positionally, and that value comes from restic's own `--json` output rather than
from a recipe.

**Secrets are kept out of argument lists where it matters.**
`Client.RunContainer` forwards environment by **name** only
(`args = append(args, "-e", k)`, `compose.go:283-290`) and passes the values through
`cmd.Env`, so `RESTIC_PASSWORD` never appears in a process listing or in the debug log.
`runEnv`'s debug line prints the argv and deliberately not the environment. Two smaller
leaks do exist by the same measure and are worth a line rather than a finding: `Exec`
puts `-e k=v` pairs into the argv it logs (`compose.go:151-154`), and an HTTP check's
`basic_auth` becomes `-u user:pass` in the helper container's argv
(`check/run.go:116-118`) - both carry only recipe-declared throwaway credentials, which
SECURITY.md explicitly puts out of scope.

**Report field hygiene.** `Observation.Body`, `Stdout`, `Stderr` and `Summary` are
`json:"-"` (`check/expect.go:29-32`), so an HTTP response body from the restored
application is used for matching and then dropped rather than serialised. The TTY report
references none of them (`grep -n 'Body\|Stdout\|Stderr\|Logs' internal/report/tty.go`
returns nothing). That is a deliberate control and it works.

**Hint rules are printed, never executed.** `Rule.RenderCommands`
(`hints/hints.go:148-172`) renders with `Option("missingkey=error")` against a two-key
context and returns strings; nothing in the tree executes them. `--hints` loads an
additional catalog with `KnownFields(true)` and compiles every regexp at load time.

**Recipe templating is closed.** `TemplateContext.Render`
(`recipe/template.go:83-96`) uses `text/template` with `missingkey=error` and a
two-function `FuncMap` (`quote`, `default`) against a fixed three-key data map. There is
no way to reach the filesystem, the environment or an arbitrary method from a recipe
template.

**YAML tags are refused before anything parses the document.** `RejectYAMLTags`
(`recipe/load.go:312-346`) runs first in both `recipe.parse` and `safety.Parse`, tracks
single quotes, double quotes and comments, and rejects a `!` in value position.

**Workspace teardown.** Registered with `defer` before the first resource exists in both
`runner.Run` (`runner.go:128-155`) and `harness.stageB` (`stageb.go:70-90`), runs on
panic, closes the log handle before removing the directory, and retries `RemoveAll` five
times for the Windows lock case. `Relax` and `Sanitise` both refuse a root outside the
workspace, and `TestRelaxRefusesToLeaveTheWorkspace` /
`TestSanitiseRefusesToLeaveTheWorkspace` cover it.

**Static analysis is clean.**

```text
$ go vet ./...            (no output)
$ golangci-lint run
0 issues.
$ go test ./...
ok  ...internal/check ...internal/cli ...internal/harness ...internal/hints
ok  ...internal/loader ...internal/nudge ...internal/recipe ...internal/recipe/safety
ok  ...internal/report ...internal/source/restic ...internal/workspace
```

**Actions are pinned by commit SHA.** Every `uses:` in all four workflows carries a
40-character SHA with the tag in a trailing comment (`actions/checkout`,
`actions/setup-go`, `actions/upload-artifact`, `golangci/golangci-lint-action`,
`goreleaser/goreleaser-action`). `permissions: contents: read` is set at workflow level on
all four; the two jobs that need more (`release`, `recipe-health.health`) raise it at job
level only. `dependabot.yml` covers both `gomod` and `github-actions` weekly. Recipe jobs
run on `pull_request`, never `pull_request_target`. All of that is right.
