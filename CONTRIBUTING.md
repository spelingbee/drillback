# Contributing to restored

The one number this project is trying to move is **the number of different people with
a merged pull request**. Not stars, not features. Everything below is arranged around
making your first one cheap.

If something here is wrong, out of date, or slower than it says, that is a bug worth
reporting on its own.

---

## Add a recipe in 10 minutes

A recipe teaches `restored` how to stand one application up from a backup and how to
tell whether the restore actually worked. It is two YAML files. You do not need to
know Go, and you do not need to know anything about `restored`'s internals.

### 0. Get the tool (2 minutes)

You need **Docker** with compose v2, **restic**, and **Go 1.27** (until there are
released binaries).

```sh
git clone https://github.com/spelingbee/restored
cd restored
go build -o bin/restored ./cmd/restored
./bin/restored version
```

### 1. Start from the compose file you already have (1 minute)

```sh
./bin/restored recipe init myapp --compose ~/docker/myapp/docker-compose.yml
```

That reads your compose file and writes `recipes/myapp/`: your volumes become inputs,
a PostgreSQL or SQLite service becomes the right kind of database input, the container
side of your published port becomes the ready probe, and everything it could not
decide is a `TODO` that names the decision. Published ports, host bind mounts and
references to your own environment are all removed, because a recipe runs on a
stranger's machine.

No compose file? Copy the skeleton instead:

```sh
cp -r recipes/TEMPLATE recipes/myapp
```

### 2. Fill in the TODOs (5 minutes)

Open `recipes/myapp/recipe.yaml`. Four sections matter.

**`inputs`** — the logical parts of a backup, named by what they are rather than by
where they live. You choose the names and guess the paths; a user whose layout differs
runs `--input data=/their/path` and nothing else changes.

```yaml
inputs:
  data:
    kind: dir              # dir | sqlite | postgres-dump
    title: The application data directory
    description: >
      What it holds, and where a default install keeps it. Somebody reads this while
      trying to work out which of their directories it is.
    default_path: /srv/myapp/data
    required: true
    mount:
      env: RESTORED_INPUT_data
      into: app:/data      # service:path
```

**`ready`** — retried probes that answer one question: has the application come up at
all? Keep them cheap. A probe that reads the database is a check wearing the wrong
hat, and it turns a data problem into "the app never started".

**`checks`** — assertions about the restored application. They run once, in order, and
every one runs even after an earlier one has failed, so the report shows every way the
restore fell short rather than only the first.

> **At least one check must FAIL against an empty application.**
>
> This is not a style rule. `restored recipe test` starts your stack with empty inputs
> and rejects the recipe if everything passes, because a recipe whose checks pass
> against an empty database manufactures confidence — which is worse than having no
> recipe at all.

Ask of every check: *would this still pass if the backup were empty?* A row count, a
file the user created, an API listing that is empty on a fresh install — those are the
checks that earn the recipe. A home page that renders is not.

Watch for rows the application creates for itself. A fresh Paperless-ngx makes two user
accounts before anybody signs up, so `count(*) FROM auth_user` passes against nothing;
that recipe excludes them by name and says so in a comment. Find that out by running
stage A and looking, not by guessing.

**`test`** — what drives the round trip. `seed` creates data the way a user would;
`export` writes out what a backup would have taken. Prefer the application's own API or
CLI over writing to its database: a recipe that seeds through the front door proves the
restore end to end and keeps working when the schema moves. Where that is not practical
— a setup wizard driven over a websocket, say — seeding the database directly is
accepted, but **say so in a comment on the step**.

Then open `recipes/myapp/compose.yaml`. The rules, all mechanically enforced:

- no `ports:` — nothing is published, and checks run from a helper container on the
  run's internal network;
- no `privileged:`, `network_mode:`, `pid:`, `ipc:`, `userns_mode:`, `devices:`,
  `cgroup_parent:`, `build:`, `extends:`, `external_links:`;
- every bind mount's source is a `${RESTORED_*}` placeholder, so it resolves inside the
  run workspace and nowhere else;
- the network is `internal: true`;
- every image carries a tag, and a digest is better.

These are the reason it is safe to run somebody else's backup, so there is no exception
process. If a recipe seems to need one, that is a finding worth an issue.

### 3. Run what CI runs (2 minutes, or twenty)

```sh
./bin/restored recipe validate ./recipes/myapp --strict
./bin/restored recipe test ./recipes/myapp
```

`recipe test` is the whole review process, mechanised:

- **Stage A, negative.** Your stack starts with empty inputs and every check runs. At
  least one must fail. If they all pass you get
  `recipe has no data-sensitive check: add a check that depends on restored data`,
  and exit 2.
- **Stage B, positive.** A fresh stack starts, `test.seed` runs, `test.export` runs,
  the result goes into a throwaway restic repository, everything is torn down, and then
  an ordinary `restored check` runs against that repository. Every check must pass.

Stage B ends by running the command a user runs. There is no test-only restore path, so
the harness cannot pass while the real one is broken.

Useful while you are working:

```sh
./bin/restored recipe test ./recipes/myapp --stage a          # just the negative half
./bin/restored recipe test ./recipes/myapp --keep             # leave the stack up to poke at
./bin/restored recipe test ./recipes/myapp --timeout 30m      # a slow first-run migration
./bin/restored recipe test ./recipes/myapp --log-level debug  # every command, on stderr
```

With `--keep` you get a workspace path and a compose project name; `docker compose -p
<project> exec <service> sh` from there.

### 4. Write the README (2 minutes)

Every recipe ships one, and it answers three questions — see
[recipes/TEMPLATE/README.md](recipes/TEMPLATE/README.md) for the shape, and
[recipes/nextcloud/README.md](recipes/nextcloud/README.md) for a filled-in example.

### 5. Open the pull request

```sh
git switch -c recipe-myapp
git add recipes/myapp
git commit -m "recipes: add myapp"
git push -u origin recipe-myapp
gh pr create
```

If `restored check` already printed you a prefilled GitHub link after a successful run,
that link opens the file-creation editor with your `recipe.yaml` already in it. It
carries only `recipe.yaml`, so you will still need to add `compose.yaml` — for anything
non-trivial, the fork-and-branch route above is less work, not more.

---

## What CI does to your pull request

| workflow | when | what it does |
|---|---|---|
| `ci.yml` | every PR | gofmt, `go vet`, golangci-lint, the English-only check, the unit suite on Linux/macOS/Windows, and the integration suite |
| `recipes.yml` | PRs touching `recipes/**` | works out which recipes changed and runs `restored recipe test` on each, in a matrix, one verdict per recipe |
| `recipe-health.yml` | weekly | runs every recipe and opens an issue when one breaks |

**A pull request that touches only `recipes/<name>/**` needs only `recipes.yml` to be
green.** You are not expected to make the Go test suite pass to add a recipe.

A change to `internal/**`, `schema/**` or `docs/hints.yaml` runs *every* recipe, because
those are the things that can break all of them at once.

Recipe jobs are capped at 15 minutes each. A recipe that cannot round-trip in that time
is out of scope for now, and its README should say so.

---

## Review promise

- **First response within 24 hours.** Not necessarily a merge — an answer.
- **Merged within 48 hours when CI is green**, for a recipe that has a data-sensitive
  check, pinned images, and a README.

If a recipe's round trip passes, a maintainer does not need to understand your
application to trust it. That is the entire point of the harness, and it is why the
review promise can be that short.

If those windows slip, say so on the pull request. A slow review is a bug in this
project's process, not a fact of life.

---

## Other things worth contributing

### A hint

`docs/hints.yaml` maps error text to an explanation and a command. It is the cheapest
useful contribution here: if you hit a confusing restore failure, the fix is often
twenty lines in that file.

```yaml
  - id: postgres/role-missing
    match: 'role "([^"]+)" does not exist'
    title: The dump references a database role that does not exist here
    text: >
      ...what happened, and what to do about it.
```

Rules are matched in order, first match wins, at most one hint per run. Add a fixture
to `internal/hints/hints_test.go` with the real error text you saw.

### A check type

The `expect` vocabulary is deliberately small — small enough for a reviewer to hold in
their head — and it is a closed list rather than an expression language. Adding to it
is a real change with a real cost, so open an issue first describing the recipe you
cannot write without it.

### A source

`restored` reads restic repositories and already-restored directories. Borg, kopia,
rsnapshot and plain tarballs are all missing. A source implements the small interface
in `internal/source`, and `internal/source/dir` is 40 lines if you want to see the
shape.

### A notifier

Right now the exit code is the whole notification story: `1` says the backup is broken,
`2` says the drill is broken, and a cron line can act on that. A notifier that posts to
ntfy, Discord or a webhook is on the roadmap and not yet designed.

---

## Working on the Go code

```sh
go build ./...
go test ./...                          # green with NO docker and NO restic installed
go test -tags integration ./... -timeout 30m   # needs docker
gofmt -l .                             # must print nothing
go vet ./...
golangci-lint run
./scripts/lint-english.sh
```

Conventions, in short:

- **Commit after every green step.** A commit is the unit of recoverable work: commit
  when something builds and its tests pass, not when a feature is finished. Never
  commit red.
- **Comments explain *why*.** The code already says what.
- **Errors are wrapped with context** and never discarded.
- **English only**, everywhere — code, comments, docs, commit messages, fixture names.
  CI enforces it.
- **No new dependency** without a line in `DECISIONS.md` justifying it. The budget is
  small on purpose.
- **Never hand-write demo output.** Anything that claims to show what the tool prints
  is captured from a real run by `scripts/capture-demo.sh`.

Architectural decisions live in [DECISIONS.md](DECISIONS.md). Fifty-odd of them are
already made; read the relevant one before proposing a change to it, and add a new ADR
rather than quietly reversing an old one.

---

## Reporting a security issue

Do not open a public issue. See [SECURITY.md](SECURITY.md).

## Code of conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).
