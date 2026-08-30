# Maintainer review - restored

Reviewed 2026-08-30 at commit `d5c2f6c2d1fa8e5fff0fb5315f1e707604db4365` (branch `main`, clean tree).
Inspected: `CLAUDE.md`, `CONTRIBUTING.md`, `README.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CODEOWNERS`,
all of `.github/` (4 workflows, 4 issue templates + chooser, PR template, dependabot), all of `scripts/`,
`recipes/TEMPLATE`, `recipes/README.md`, `DECISIONS.md`, `PROGRESS.md`, `SPEC.md` sections 7, 8 and 12, and the
Go paths that produce contributor-facing output (`internal/harness`, `internal/report`, `internal/recipe/safety`,
`internal/nudge`, `internal/cli`). Ran `go build`, `go test ./...`, `restored recipe init --compose`,
`restored recipe validate` (five variants), the `tools/gen` generators, and a local `git` experiment against the
`select` job's diff logic. No Docker was started; the round-trip paths were read, not run.

## Summary

| severity | count |
|---|---|
| P0 | 3 |
| P1 | 5 |
| P2 | 7 |
| P3 | 3 |
| **total** | **18** |

The contributor machinery here is better than most projects have at 1.0, and the central bet - that a passing
round trip lets a maintainer merge without understanding the contributor's application - is real and mostly
delivered. But it is not ready to receive strangers today, for three reasons. First, the isolation that makes
the bet safe is enforced by a deny-list over `services.*` only: a `compose.yaml` can bind the host root
filesystem into a container through the top-level `volumes:` key and `restored recipe validate --strict` prints
`ok` (MNT-01). Because `assets.go` embeds `all:recipes` into the binary, one merged recipe is shipped to every
user, so this is a supply-chain hole and not just a CI hole. Second, the very first recipe PR anyone opens will
show a red `ci / generated` job, because adding `recipes/<name>/` changes two checked-in generated files, and
`CONTRIBUTING.md:180` promises in bold that a recipe-only PR needs nothing but `recipes.yml` (MNT-02). Third,
when a stranger's round trip fails, the CI log and the uploaded artifact contain one sentence and no per-check
detail - the inner `report.Report` with the query, the expectation, the actual value, the hint and the service
logs is computed and then dropped on the floor (MNT-03). The README's failure output is what `restored check`
prints; it is not what a contributor sees on their PR.

What breaks first at 10 PRs a week is not throughput - the harness genuinely removes most of the judgement. It
is MNT-03 plus MNT-09: every failing recipe PR becomes a manual translation job for the maintainer ("run it
locally, read the real output, explain it in a comment"), and every incoming feature issue needs a label applied
by hand because four of the ten labels the script creates are never applied by anything. That is the per-PR
time that does not scale. Second to break is the review promise: `CONTRIBUTING.md:193` says first response
within 24 hours, but on a public repository a first-time contributor's workflow run does not start until the
maintainer clicks Approve, so the clock the contributor experiences is the maintainer's clock, and nothing in
the repository says so (MNT-06).

## The 10-minute path, timed

Walked as a newcomer who runs a self-hosted PostgreSQL-backed application on Linux, has Docker, and has never
seen this repository. Times are hands-on minutes.

| # | step | what the contributor does | realistic minutes | must read first |
|---|---|---|---|---|
| 0 | find the docs | README badge block -> "Add a recipe in 10 minutes" -> `CONTRIBUTING.md` | 5 | `README.md` (1,799 words), `CONTRIBUTING.md` sections 0-5 (~900 of its 1,832 words) |
| 1 | prerequisites | install Go 1.27 and restic; Docker assumed present | 8 (CONTRIBUTING claims 2, and counts only the clone) | `CONTRIBUTING.md:18-28` |
| 2 | clone and build | `git clone`, `go build -o bin/restored ./cmd/restored` - first build resolves the module cache | 2 | - |
| 3 | `recipe init --compose` | one command; output is genuinely good, naming the app service, the port, each dir input and the database | 1 | `CONTRIBUTING.md:30-47` |
| 4 | learn the vocabulary | recipe vs compose, 3 input kinds, `mount.into` service:path syntax, `within`, ready-vs-check, the 18 `expect` keys, `{{ .vars.x }}` and `{{ .inputs.db.path }}`, the empty-stack rule | 15 | `docs/recipe-spec.md` (1,399 words), `recipes/TEMPLATE/recipe.yaml` (1,431 words), one worked recipe |
| 5 | fill the 9 TODOs in `recipe.yaml` | the hard one is the data-sensitive check: name a table a fresh install leaves empty, and exclude the rows the app creates for itself | 20 | `CONTRIBUTING.md:80-94`, the application's own schema |
| 6 | **write `test.seed`** | `recipe init` leaves this `[]`. Create real data through the app's own API, from inside a network with no published ports, no host access and no browser: bootstrap an admin, get past a first-run wizard, handle auth or CSRF. Each attempt costs a full container cycle | **25-90** | `CONTRIBUTING.md:96-101`, `SPEC.md 7.3`, `recipes/gitea/recipe.yaml` |
| 7 | `recipe validate --strict` | fails immediately: `recipe init` writes `maintainers: []` and `--strict` promotes that warning to exit 2 (MNT-10) | 1 | - |
| 8 | first `recipe test` | pulls the app image, `postgres`, `curlimages/curl`, `restic/restic:0.19.1`; several hundred MB | 6 | `CONTRIBUTING.md:117-148` |
| 9 | the debug loop | 3-8 iterations of steps 6 and 8; bundled recipes run 27s-2m34s each once cached | folded into 6 | - |
| 10 | write `recipes/<name>/README.md` | three required sections, including "which of my directories is this input" for two or three real deployment shapes - genuine research about other people's setups | 15 (CONTRIBUTING claims 2) | `recipes/TEMPLATE/README.md`, `recipes/nextcloud/README.md` |
| 11 | regenerate the index | `go run ./tools/gen recipes-index > recipes/README.md` and `go run ./tools/gen readme-table` - **documented nowhere in the recipe path** | 0, or 25 of confusion after CI goes red (MNT-02) | nothing tells them |
| 12 | branch, commit, push, PR | plus the PR template: 7 checkboxes, paste the real output, state the round-trip time | 8 | `.github/PULL_REQUEST_TEMPLATE.md` |
| 13 | wait for CI | first-time contributor: the run does not start until a maintainer approves it; then 4-8 min of runner time | unbounded + 6 | nothing tells them (MNT-06) |

**Honest total: 105 to 180 minutes of hands-on work for a first recipe**, of which 15-20 is one-time toolchain
setup. The ten-minute claim is defensible for exactly one thing, and it is worth keeping: `recipe init --compose`
to a first draft that parses really is about ten minutes, and that command is the best thing in the project.
It is not the time to a mergeable pull request, and `README.md:241` and `CONTRIBUTING.md:12` both present it as
the latter.

**The single biggest time sink is `test.seed`** (step 6). It is the one part `recipe init` cannot draft, the one
part with no worked reference beyond reading two other recipes, the one part that must run inside an internal
network with nothing published, and the one part where every attempt costs a full compose up/down. Gitea got
away with two steps because it has an admin CLI and a clean REST API; `PROGRESS.md` records that Nextcloud
needed a bespoke twenty-line `prepare` service. A contributor whose application has a first-run wizard is
looking at an afternoon, and nothing in `CONTRIBUTING.md` warns them before they start.

Second-biggest, and cheap to fix: step 10. `CONTRIBUTING.md:149-153` budgets two minutes for a README that
`recipes/TEMPLATE/README.md:23-39` correctly demands be researched.

## Findings

### MNT-01 (P0) `recipe validate --strict` accepts a bind mount of the host root filesystem

**Where:** `schema/compose-safety.schema.json:5-83` (root `properties` covers only `services` and `networks`;
there is no `additionalProperties: false` and no constraint on the top-level `volumes`, `configs` or `secrets`
keys) / `internal/recipe/safety/interpolate.go:103-136` (`CheckResolvedMounts` inspects only
`services.*.volumes` short-syntax strings, and `isHostPath` at line 158 returns false for a named volume) /
`internal/recipe/safety/safety.go:265-277` (`forbiddenService` is a deny-list over service keys).

**What:** The isolation rules are enforced against `services.*` only. Three top-level compose keys are not
validated at all, and each of them reintroduces an absolute host path:

```yaml
volumes:
  hostroot:
    driver: local
    driver_opts: { type: none, device: /, o: bind }
```

referenced from a service as `- hostroot:/host` passes the short-syntax pattern at
`schema/compose-safety.schema.json:48` because `hostroot` is a legal named volume, and `CheckResolvedMounts`
skips it for the same reason. The same is true of `configs:` and `secrets:` with a `file:` source. Verified on
this commit, using `recipes/uptime-kuma/recipe.yaml` with a substituted `compose.yaml`:

```text
$ ./bin/restored recipe validate /tmp/tmp.3uQZBj8Tkc/evil3 --strict
ok       C:/Users/kadyr/AppData/Local/Temp/tmp.3uQZBj8Tkc/evil3
exit=0

$ ./bin/restored recipe validate /tmp/tmp.n0wdxzX9ag/evil2 --strict   # configs: hostfile: {file: /etc/passwd}
ok       C:/Users/kadyr/AppData/Local/Temp/tmp.n0wdxzX9ag/evil2
exit=0
```

For contrast, the deny-list does work where it is written: `cap_add: [SYS_ADMIN]` and
`security_opt: [apparmor:unconfined]` are both correctly rejected by the same command.

This contradicts four documents. `CONTRIBUTING.md:109-110`: "every bind mount's source is a `${RESTORED_*}`
placeholder, so it resolves inside the run workspace and nowhere else". `README.md:209-213`: "no bind mount
outside the run's own workspace ... `restored recipe validate` rejects a recipe that breaks any of it".
`SECURITY.md:31-39` puts exactly this in scope ("Anything that lets a recipe, a compose file ... read, write,
or delete a path outside `<tmp>/restored-<runid>`", and "A malicious recipe from a pull request doing any of
the above while passing `recipe validate` and `recipe test`"). `PROGRESS.md:38` asserts "Isolation | enforced:
... no bind outside the workspace" with no command behind it, which is the failure mode `CLAUDE.md` section
*Evidence, not assertion* exists to prevent.

The blast radius is larger than CI. `assets.go:18` embeds `all:recipes` into the binary, so every merged recipe
ships to every user and runs on their host, where the mount is read-write.

**Scenario:** Dana opens a PR adding `recipes/immich/`. Its `compose.yaml` declares a named volume with
`driver_opts: {type: none, device: /, o: bind}` mounted at `/host` in the seeder, and a `test.seed` `exec` step
that copies `/host/home/*/.ssh/id_*` into `$RESTORED_EXPORT`. `recipe validate --strict` prints `ok`,
`recipe test` passes both stages, `recipes.yml` is green, the PR satisfies every line of the PR template, and
the maintainer merges it in 40 minutes exactly as the review promise says they can. It is then embedded in the
next binary and runs on the host of everyone who does `restored check --recipe immich`.

**Proposed fix:** Make the compose safety schema an allow-list rather than a deny-list. At minimum, add to the
root `"volumes": {"additionalProperties": {"type": ["object","null"], "not": {"required": ["driver_opts"]}}}`
and forbid the top-level `configs` and `secrets` keys outright (a recipe has no legitimate use for either -
`environment:` covers it, and the throwaway credentials are literals by design per `SECURITY.md:68-72`). Then
extend `CheckResolvedMounts` to walk the resolved top-level `volumes` map and reject any `driver_opts.device`,
so the runtime half of the rule matches the static half. Add a fixture per shape to
`internal/recipe/safety/safety_test.go`. Longer term, `unevaluatedProperties: false` on the service object with
an explicit allow-list of compose keys is the only version of this that stays correct as compose grows.

### MNT-02 (P0) A recipe-only pull request fails `ci / generated`, which CONTRIBUTING promises it will not

**Where:** `CONTRIBUTING.md:180-181` ("**A pull request that touches only `recipes/<name>/**` needs only
`recipes.yml` to be green.** You are not expected to make the Go test suite pass to add a recipe.") /
`.github/workflows/ci.yml:10-13` (`on: pull_request`, no `paths` filter) / `.github/workflows/ci.yml:65-75`
(the `generated` job regenerates and diffs `docs/recipe-spec.md`, `recipes/README.md` and `README.md`) /
`assets.go:18` (`//go:embed all:recipes`, so a new directory joins the registry automatically).

**What:** Adding `recipes/<name>/` changes two checked-in generated files, and `ci.yml` fails on the diff.
Verified by copying a scaffolded recipe into `recipes/myapp` on this commit:

```text
$ go run ./tools/gen recipes-index | diff recipes/README.md -
15a16
> | [`myapp`](myapp/) | Myapp | directories + PostgreSQL | 3 | - |
21c22
< 5 recipes.
---
> 6 recipes.

$ go run ./tools/gen readme-table && git diff --stat -- README.md
 README.md | 1 +
 1 file changed, 1 insertion(+)
```

So `ci / generated` goes red on every recipe PR. The walk-through never mentions regenerating: step 4 is "write
the README", step 5 is "open the pull request". The PR template has no line for it either. The CI error message
is good (`The generated files are out of date. Run: make docs recipes-index`) but it points at `make` and at Go
tooling, immediately after `CONTRIBUTING.md:16` promised "You do not need to know Go".

**Scenario:** Sam, whose only interaction with this project is one Miniflux recipe, pushes a green
`recipes.yml` and a red `ci`. `CONTRIBUTING.md` told them in bold that `ci` does not apply to them. They either
open a "is this me?" comment - one maintainer round trip that the whole design exists to avoid - or they assume
the project is broken and close the tab. At 10 PRs a week that is 10 avoidable comments a week.

**Proposed fix:** Either (a) generate `recipes/README.md` and the README table in CI and commit them from a
follow-up job on `main`, dropping them from the PR-time diff; or (b) keep the diff, add
`go run ./tools/gen recipes-index && go run ./tools/gen readme-table` as step 4.5 of the walk-through and a
checkbox in the recipe section of the PR template, and delete the claim at `CONTRIBUTING.md:180`. Option (a)
matches the project's own tie-break rule ("a scaffold that generates the right thing, over documentation
explaining the right thing").

### MNT-03 (P0) A failing round trip in CI produces one sentence, and the detail that would fix it is discarded

**Where:** `internal/harness/harness.go:96-119` (`Stage` and `Result` carry `Reason string` and `Error string`
and nothing else - no field holds the inner `report.Report`) / `internal/harness/stageb.go:264-276` (the inner
report `rep` is read for two counters and then goes out of scope) / `internal/harness/render.go:80-118`
(`writeResult` prints status, reason, phase timings and one command) / `internal/cli/recipetest.go:97-101`
(`--report` writes the harness report, which has no checks in it) / `.github/workflows/recipes.yml:142-151`
(the artifact is `recipe-test.log` and `recipe-test.json`).

**What:** `internal/report/report.go:28-44` shows what the inner run computes: `Checks`, `Logs` (per-service log
lines), `Hint`, `Warnings`, `Inputs` with byte counts. That is the entire content of the RESTORE UNUSABLE block
in `README.md:60-113`, the thing the project sells. During `recipe test` none of it is rendered and none of it
is serialised. What a contributor gets instead is the string built at `internal/harness/stageb.go:294-296`:

```text
2 of 5 checks failed after a real round trip (repos-in-db, api-lists-repos): the seed,
the export or the check disagree about where the data lives
```

That names which checks failed. It does not say what the query was, what was expected, what came back, or what
the application logged. The `--log-level debug` stream that `recipes.yml:137` enables is docker and restic
command lines plus their stderr (`internal/compose/compose.go:80-88`), not check results. The per-run
`debug.log` (`internal/runner/runner.go:109`) is written into the workspace, and the workspace is removed at
`internal/harness/stageb.go:87` because CI does not pass `--keep`.

Worse, the one actionable line that is printed - `st.Command`, set at `internal/harness/stageb.go:234` to
`restored check --recipe <dir> --source restic --from <workspace>/repo --snapshot latest` - names a directory
the same function has just deleted. Pasting it gets "no such file or directory".

**Scenario:** Priya's Miniflux recipe passes stage A and fails stage B on one check. She opens the run, reads
"1 of 4 checks failed after a real round trip (entries-in-db)", downloads the artifact, finds the JSON has
`{"name":"B","status":"fail","reason":"..."}` and no check array, copies the reproduction command, and gets
ENOENT. She now has to reproduce locally with Docker to learn what she could have been told - or she asks the
maintainer, who reproduces it locally instead. That is the review burden the harness was built to remove,
reappearing on every failing PR.

**Proposed fix:** Add `Check *report.Report` (or the trimmed `Checks`, `Logs`, `Hint` triple) to
`harness.Stage`, populate it from `rep` in both `stageA` and `stageB`, serialise it in the harness JSON, and
have `writeResult` call the existing `report.Report.WriteTTY` renderer indented under a failed stage. That
renderer already exists, is pure, and is golden-tested (`internal/report/testdata/unusable.txt`). Separately,
set `st.Command` to something that will still work - `restored recipe test ./recipes/<name> --stage b --keep`
- rather than to a path that has been removed.

### MNT-04 (P1) The one-click contribution nudge produces a pull request that always fails validate

**Where:** `internal/nudge/nudge.go:51-56` ("Adding it is one click:" plus a
`github.com/.../new/main?filename=recipes/<name>/recipe.yaml&value=...` link) / `SPEC.md 8.2` (the URL carries
`recipe.yaml` only) / `CONTRIBUTING.md:165-168` (acknowledges the gap in the walk-through, where the nudge's
audience is not).

**What:** The link creates a branch containing `recipes/<name>/recipe.yaml` and nothing else. `recipes.yml`
selects that directory, and the first step of the matrix job is
`restored recipe validate ./recipes/<name> --strict`, which cannot pass:

```text
$ ./bin/restored recipe validate /tmp/tmp.Lf91eQl1p9/onlyrecipe --strict
INVALID  .../onlyrecipe
         reading compose.yaml for recipe "gitea": open .../onlyrecipe/compose.yaml:
         The system cannot find the file specified.
exit=2
```

`SPEC.md 8.1` condition 5 exists precisely so the nudge never sends someone at a PR CI will reject
("nudging someone toward a PR that CI will immediately reject wastes their goodwill, which is the scarcest
resource this project has"). The condition checks the recipe; it does not check that the resulting pull request
is complete. This is the highest-volume acquisition surface in the project - it fires after every passing run of
every local recipe - and it lands on a guaranteed red X.

**Scenario:** Tomas runs `restored check --recipe ./my-linkwarden` against his own backup, it passes, he clicks
the link, GitHub opens the file editor with his recipe in it, he clicks "Propose new file". `recipes.yml`
reports INVALID with a message about a file he was never shown. His first contact with the project is a
rejection for something the project told him was one click.

**Proposed fix:** Cheapest correct change: alter the nudge text so it does not say "one click" for a path that
needs two files. Print the fork-and-branch four-step block (which `internal/nudge/nudge.go:61-65` already
composes for the oversize case) as the primary path, and demote the prefilled link to "or paste just the recipe
here, and add `compose.yaml` on the branch GitHub creates for you". Alternatively, have `recipes.yml`'s `select`
step detect a recipe directory with no `compose.yaml` and report "this PR needs `compose.yaml` too" instead of
INVALID.

### MNT-05 (P1) PASS-BY-STARTUP-REFUSAL means stage A proved nothing, and nothing mechanical takes over

**Where:** `internal/harness/harness.go:326-332` (`case len(rep.Checks) == 0:` sets `st.Status = StatusPass`) /
`SPEC.md 2044-2048` (which calls the same outcome "INCONCLUSIVE, not a failure") / `DECISIONS.md ADR-032` /
`.github/PULL_REQUEST_TEMPLATE.md:19-21` / `README.md:268-270` and `CONTRIBUTING.md:197-199`.

**What:** When the empty stack never becomes ready inside 90 seconds, stage A passes with zero checks executed.
`ADR-032`'s consequences are honest about it ("a maintainer reviewing a recipe can see that the checks
themselves were never exercised negatively - which is weaker evidence, and should look weaker"). But the
mechanism that is supposed to replace the evidence does not exist. Consider a recipe whose application
crash-loops on a zero-length SQLite file but starts fine on a schema-only dump, and whose four checks are all
`http status: 200` and `file exists: true`:

- stage A: the app never starts, zero checks run, **PASS** by refusal;
- stage B: real data, the app starts, all four checks pass, **PASS**;
- `recipes.yml`: green;
- against a real backup whose dump was taken `--schema-only`: the app starts, all four checks pass, **PASS**.

That is the false PASS the entire tool exists to destroy, shipped with a green badge on it. It is not
hypothetical territory: `PROGRESS.md:576` records `uptime-kuma`, one of the five shipped recipes, taking exactly
this path.

Meanwhile `README.md:268-270` and `CONTRIBUTING.md:197-199` state the merge-without-understanding promise
unconditionally, with no carve-out.

**Scenario:** Aleks contributes a Firefly III recipe. Stage A reports PASS-BY-STARTUP-REFUSAL, stage B passes,
`recipes.yml` is green, and the PR template's checkbox 2 is ticked with the note "the app refuses to boot with
no database". The maintainer, following the promise in `CONTRIBUTING.md`, merges without reading the four
checks. None of them touches a row.

**Proposed fix:** Two changes. (1) In `harness.stageA`, when `len(rep.Checks) == 0`, keep the stage passing but
require at least one check whose `kind` is `sql` or whose `expect` contains `scalar_int_min`,
`json_path_len_min` or `glob_min_count` - the closed `expect` vocabulary makes that a mechanical test, which is
the whole reason `README.md:205-207` says the vocabulary is closed. Reject with a message in the same shape as
`NoDataSensitiveCheck`. (2) Add the carve-out to `CONTRIBUTING.md:197` in one sentence, so the promise stays
true.

### MNT-06 (P1) The 24-hour review promise runs on the maintainer's clock, and there is no aggregate CI gate

**Where:** `CONTRIBUTING.md:191-202` (the review promise) / `.github/workflows/recipes.yml:11-25`,
`:96-110` (the `test` matrix, skipped entirely when `mode == none`) / `PROGRESS.md:783-797` (the launch
checklist).

**What:** On a public repository GitHub's default Actions setting is "Require approval for first-time
contributors". Every first-time contributor's `recipes.yml` and `ci.yml` runs sit pending until the maintainer
clicks Approve. That setting is correct and should stay on - `recipes.yml` pulls and runs images a stranger
names, and see MNT-01 - but it means:

- the contributor's PR shows *no checks at all*, not a running check, which reads as "nothing happened";
- "Merged within 48 hours when CI is green" cannot start until the maintainer is already at the keyboard, which
  removes most of the value the promise is trying to buy;
- nothing in `CONTRIBUTING.md`, the PR template, or the launch checklist mentions the setting, so the first time
  a maintainer meets it will be on somebody's pull request.

Two related gaps in the same workflow. `recipes.yml` has no aggregate job: `test` is a matrix producing
`test (gitea)`, `test (nextcloud)`, and so on, and when nothing is selected (`mode: none`) it is skipped
entirely, so there is no stable job name to require in branch protection. Whether a recipe PR is mergeable is
therefore an eyeball judgement on the checks tab. And `recipes.yml:127-129` is correctly written to need no
secrets, which is the right call and worth stating in `CONTRIBUTING.md`, where a contributor working from a fork
will look for it.

**Scenario:** Jo, who has never contributed here, opens a recipe PR at 09:00. No checks appear. At 09:40 they
comment "did I do something wrong?". The maintainer, who intended to respond within 24 hours anyway, spends the
response on the CI settings instead of on the recipe.

**Proposed fix:** Add a `verdict` job to `recipes.yml` with `if: always()`,
`needs: [select, test, test-sequential]`, failing if any needed job failed and succeeding when none was
selected; make that the single required status check. Add three lines to `CONTRIBUTING.md`'s "What CI does to
your pull request": that a first PR waits for a maintainer to approve the run, that this is deliberate, and that
no secrets are needed so a fork behaves exactly like a branch. Add "check the Actions approval setting" to the
launch checklist in `PROGRESS.md`.

### MNT-07 (P1) Both reporting channels point at a repository setting nobody has enabled, and conduct reports go to the accused

**Where:** `SECURITY.md:7-9` / `CODE_OF_CONDUCT.md:38-40` / `.github/ISSUE_TEMPLATE/config.yml:13-16` /
`PROGRESS.md:783-797`.

**What:** All three route to `https://github.com/spelingbee/restored/security/advisories/new`. Two problems.

First, GitHub private vulnerability reporting is off by default and must be enabled per repository. The launch
checklist at `PROGRESS.md:783-797` lists creating labels, making the repository public, the recipes-wanted
issues and the release checklist - not this. If the box is not ticked, the only advertised security channel and
the only advertised conduct channel both 404 the moment the repository goes public, and no fallback address
appears anywhere in the repository. `SECURITY.md:15-17` promises a 72-hour first response through a form that
may not exist.

Second, `CODE_OF_CONDUCT.md:39-40` sends conduct reports to a channel it says "reaches the repository owner and
nobody else", and `CODEOWNERS:4` confirms there is exactly one maintainer. A Contributor Covenant enforcement
clause whose only route is to the sole maintainer has no answer for a report about that maintainer. It is also
a category error: the security advisory form is scoped to vulnerabilities and its UI says so, which will deter
the report it is meant to receive.

**Scenario:** Two days after launch, a contributor wants to report that a maintainer comment on their pull
request was demeaning. They open `CODE_OF_CONDUCT.md`, find a link to a "Report a security vulnerability" form,
and do not file.

**Proposed fix:** Add "enable private vulnerability reporting" to the launch checklist, before "make the
repository public". Add a fallback line to `SECURITY.md` ("if that form is unavailable, email <address>") so the
policy does not depend on one setting. In `CODE_OF_CONDUCT.md`, replace the advisory link with a real contact -
an email address is the conventional answer - and add the standard sentence that a report concerning the
maintainer may be sent to GitHub Support instead.

### MNT-08 (P1) Discussions is the only question channel, is linked before it exists, and CONTRIBUTING never mentions it

**Where:** `.github/ISSUE_TEMPLATE/config.yml:3-7` (the first entry in the chooser, above the recipe guide) /
`CONTRIBUTING.md` (no occurrence - `grep -rn -i discussion` over the repository returns only `config.yml`) /
`PROGRESS.md:783-797`.

**What:** GitHub Discussions is off by default and must be enabled per repository. The issue chooser's first and
most prominent entry points at `/discussions`, which 404s until it is turned on, and enabling it is not on the
launch checklist. Independently, `CONTRIBUTING.md` has no "where to ask a question" section at all, so a
contributor who arrives through the README rather than through "New issue" never learns the channel exists.
`CONTRIBUTING.md:7-8` invites bug reports about the guide being wrong, but names no place to ask a question that
is not yet a bug.

**Scenario:** Rin is halfway through step 6 and cannot work out how to seed an app that requires a setup wizard.
`CONTRIBUTING.md` offers a bug template, a feature template and a recipe-broken template, none of which fit.
They open a `Feature` issue titled "how do I seed Wallabag?" - which the maintainer must retitle, relabel and
answer, having been given none of the context the recipe-request template would have collected.

**Proposed fix:** Add "enable Discussions" to the launch checklist next to "enable private vulnerability
reporting". Add four lines at the end of `CONTRIBUTING.md`: where to ask (Discussions), what to expect (the same
24-hour first response), what belongs in an issue instead, and that "I got stuck writing a recipe" is a welcome
question rather than a failure.

### MNT-09 (P2) Four of the ten labels are created and never applied by anything

**Where:** `scripts/labels.sh:62-73` / `.github/ISSUE_TEMPLATE/bug.yml:3`, `feature.yml:3`,
`recipe-broken.yml:4`, `recipe-request.yml:4` / `.github/workflows/recipe-health.yml:140` /
`scripts/recipes-wanted.sh:171`.

**What:** Cross-checked every label name that appears anywhere in the repository against the ten `labels.sh`
creates.

Applied automatically and created - correct: `bug` (`bug.yml:3`), `enhancement` (`feature.yml:3`),
`recipe-broken` and `help wanted` (`recipe-broken.yml:4`, `recipe-health.yml:140`), `recipe` and
`good first issue` (`recipe-request.yml:4`, `recipe-health.yml:140`, `recipes-wanted.sh:171`).

Created and never applied by any template, workflow or script: **`source`, `notifier`, `hint`, `check-type`**.
Nothing is referenced but not created, which is the more dangerous direction, so the important half is right -
`gh issue create` fails on an unknown label, and `recipe-health.yml:138-141` would open no issue at all.

The four orphans are exactly the four areas in `feature.yml:32-44`'s "Which part is this about?" dropdown
(A source / A check kind, or an expect key / A notifier / Recipe format / CLI or reporting / Something else).
The contributor answers the question and the answer goes nowhere: the maintainer must read the dropdown and
apply the label by hand, on every feature issue, forever. That is exactly the shape of work the project's own
tie-break rule says to automate ("a mechanical check that lets a maintainer merge without judgement, over a
convention that requires review").

**Scenario:** In week three the maintainer wants to find every issue asking for a new backup source. There is a
`source` label, and nothing has it, because 14 feature issues were filed with the dropdown set and no label
applied.

**Proposed fix:** A small `label-from-form.yml` workflow on `issues: [opened]` that reads the rendered
`### Which part is this about?` section and applies `source`, `check-type`, `notifier` or nothing - roughly 20
lines with `actions/github-script`. If that is unwanted, delete the four labels from `labels.sh` and drop the
dropdown: an unfilled taxonomy is worse than none, because it makes searches look empty rather than absent.

### MNT-10 (P2) `recipe init --compose` writes a recipe that fails the very next command it tells you to run

**Where:** `internal/cli/recipe.go:324-329` (the "Next:" block prints
`restored recipe validate <target> --strict`) / `internal/recipe/safety/safety.go:239-243` (empty
`metadata.maintainers` is a warning, promoted to exit 2 by `--strict`) / `recipes/README.md:29` ("a draft that
already validates") / `recipes/TEMPLATE/README.md:21` ("The result validates as it comes out") /
`PROGRESS.md:34` (same claim).

**What:** Verified on this commit against a real compose file:

```text
$ ./bin/restored recipe validate /tmp/.../rec/myapp --strict
ok       .../rec/myapp
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
exit=2

$ ./bin/restored recipe validate /tmp/.../rec/myapp
ok       .../rec/myapp
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
exit=0
```

`recipes.yml:125` runs `--strict`, so that is the real gate. The generated file comments the field
`# Put your GitHub handle here. restored does not guess it.`, leaves it empty, and then hands the contributor a
command that fails. Three documents claim the opposite.

**Scenario:** Maya's first minute with the tool: `recipe init --compose` prints four confident next steps, she
runs step 3 verbatim, and gets exit 2. Nothing is broken, but the first signal the tool gives about her work is
a failure it caused.

**Proposed fix:** Either emit `maintainers: ["@TODO-your-github-handle"]` and add a `--strict` rule rejecting
that literal (a better check than the empty-list one - see MNT-13), or add `--maintainer <handle>` to
`recipe init` and print it as step 0 of the "Next:" block. Then make the three "validates as it comes out"
claims true, or remove them.

### MNT-11 (P2) Fifty `good first issue` tickets, and no way to claim one

**Where:** `scripts/recipes-wanted.sh:97-179` and `docs/recipes-wanted.txt` (50 entries) /
`PROGRESS.md:790-793` (the launch plan runs it).

**What:** The script opens one issue per application, each labelled `recipe` and `good first issue`, each ending
in a call to action. The body is genuinely good - it links the guide, gives the `recipe init --compose` command,
states the one hard question, and says what CI will do. What it does not contain is a claiming convention: no
"comment here before you start", no assignment step, no note about checking for an open pull request first.
`CONTRIBUTING.md` has no such convention either.

`good first issue` is the single most-crawled label on GitHub. Fifty of them, all self-contained, all
independent, published on the same afternoon, is precisely the setup that produces two people writing the Immich
recipe in the same week. One of them will have spent the 105-180 minutes from the table above and will get a
"sorry, someone beat you to it". This project measures distinct people with a merged PR; a duplicated recipe
loses one, and it loses the one least likely to come back.

**Scenario:** Week two. Two Immich pull requests land four days apart. Both pass. The maintainer merges the
first and closes the second, and the second contributor's only experience of the project is three hours of work
rejected for a reason that was visible from the start.

**Proposed fix:** Add a line to the issue body at `scripts/recipes-wanted.sh:120-161`: "**Comment here before
you start** and it will be assigned to you - nobody else will start it while it is assigned. If it has been
assigned for more than two weeks with no PR, it goes back in the pool." Then either assign by hand or add a
five-line `issue_comment` workflow that self-assigns. Also stage the rollout as `PROGRESS.md:790` already plans
- `--limit 5` first - and watch for duplicates before opening the other 45.

### MNT-12 (P2) CONTRIBUTING omits the licensing, commit-message and question answers a first-timer needs

**Where:** `CONTRIBUTING.md` (whole file) / `DECISIONS.md:96-97` (the decision that is never surfaced) /
`.all-contributorsrc:11` (`"commitConvention": "angular"`) / `.github/workflows/ci.yml:45-47`.

**What:** Cross-checked what a first-time contributor needs against what is written down.

Present and good: how to run tests without Docker (`CONTRIBUTING.md:249-257`, and `ci.yml:91-95` confirms the
unit job keeps that true), what gets rejected (the isolation rules at 103-115 and the data-sensitive rule at
80-94, both mechanically enforced), how long review takes (191-202), and the English-only rule (266), which
`ci.yml:49-50` really does enforce.

Missing:

- **DCO/CLA.** `DECISIONS.md:96-97` decided it - "No CLA - contributions are under the project licence by the
  DCO-free default of GitHub's terms plus the licence itself" - and that decision appears nowhere a contributor
  reads. Anyone contributing from an employer will look for it, not find it, and ask. One sentence in
  `CONTRIBUTING.md` closes it permanently.
- **Commit message format.** `CLAUDE.md` has a convention, the git history follows it (`docs:`, `fix:`, `ci:`,
  `feat:`), `.all-contributorsrc:11` declares `angular`, and `CONTRIBUTING.md` says nothing. The maintainer will
  either rewrite messages at merge time or accept drift.
- **Where to ask a question.** See MNT-08.
- **The golangci-lint version.** `ci.yml:45-47` pins `v2.13.2`; `CONTRIBUTING.md:255` just says
  `golangci-lint run`. A contributor on v1 gets findings CI does not have, or misses findings CI does.

**Scenario:** Wei's employer requires a signed CLA, or an explicit statement that none is needed, before he may
contribute. He greps `CONTRIBUTING.md`, `SECURITY.md` and `README.md`, finds nothing, and does not contribute.
That is a lost contributor who never appears in any metric.

**Proposed fix:** Four short subsections at the end of `CONTRIBUTING.md`: Licensing (quote the ADR sentence
verbatim), Commit messages (the prefix list, one line), Where to ask (Discussions, per MNT-08), and Tool
versions (golangci-lint `v2.13.2`, matching `ci.yml`).

### MNT-13 (P2) `recipe validate --strict` passes the TEMPLATE verbatim, so the placeholder loop costs a Docker cycle

**Where:** `recipes/TEMPLATE/recipe.yaml:31` (`maintainers: ["@your-handle"]`), `:159`
(`SELECT count(*) FROM todo_replace_this_table;`), `recipes/TEMPLATE/compose.yaml:23`
(`image: example/template:1.0.0`) / `internal/recipe/safety/safety.go:239-260` (`Warnings`).

**What:**

```text
$ ./bin/restored recipe validate ./recipes/TEMPLATE --strict
ok       ./recipes/TEMPLATE
exit=0
```

Every placeholder in the skeleton passes the strict validator. A contributor who copies TEMPLATE and forgets one
- the maintainer handle, the table name, the example image - gets `ok` from the one-second command and finds out
from `recipe test`, which is the several-minute Docker command, or from CI, which is the several-minute Docker
command plus a maintainer approval. That inverts the fast/slow ordering the whole two-command design is built
on.

The three checks are trivial and entirely mechanical: reject the literal `@your-handle`, reject
`todo_replace_this_table`, reject an image whose repository is `example/`. Each moves a six-minute failure to a
one-second one, and each removes a thing the maintainer would otherwise have to spot by eye.

**Scenario:** Ola copies TEMPLATE for a Grafana recipe, fills in most of it, and misses the table name in the
third check. `validate --strict` says `ok`. `recipe test` runs for four minutes and fails stage B with
"1 of 4 checks failed after a real round trip (rows-present)" - which, per MNT-03, tells her nothing about
`todo_replace_this_table`.

**Proposed fix:** Add the three literals to `safety.Warnings` (promoted to errors under `--strict`), with the
message naming the field. Add a test case per literal to `internal/recipe/safety/safety_test.go`. While there,
consider warning on any `TODO:` remaining in a `title` or `description` field, which catches the `recipe init`
output too.

### MNT-14 (P2) The all-contributors flow cannot run, and CONTRIBUTING does not mention it

**Where:** `.all-contributorsrc:1-17` / `README.md:284-295` / `CONTRIBUTING.md` (no occurrence) /
`scripts/contributors.sh`.

**What:** `.all-contributorsrc` is valid and correctly configured (`projectName`, `projectOwner`, `files`,
`contributorsPerLine`, `commit: true`), and the README carries the paired `ALL-CONTRIBUTORS-LIST` markers.
`README.md:293-295` states plainly that the bot is not installed, because installing a GitHub App is
`CLAUDE.md` stop point 4. That honesty is right; the consequence is not managed.

`commit: true` means the flow depends on the bot having write access, so nothing happens until the App is
installed. Meanwhile `CONTRIBUTING.md` never tells a contributor the mechanism exists, so nobody will ever type
`@all-contributors please add @x for code` - and the README section will be empty on the day the project asks
people to look at it. For a project whose only metric is distinct contributors, the contributor wall is that
metric made visible, and it is currently a hole in the README between two comment markers. There is also no
`CHANGELOG.md` (MNT-17), so a merged first contribution presently produces no visible record anywhere.

**Scenario:** The tenth merged recipe. Ten people have contributed, the README Contributors section is empty,
and each of them checked.

**Proposed fix:** Add installing the all-contributors App to the launch checklist in `PROGRESS.md`, next to
`labels.sh --apply`. Add three lines to `CONTRIBUTING.md` saying a merged PR earns an entry and that anyone can
trigger it with `@all-contributors please add @handle for code`. If the App will not be installed, use the
`all-contributors` CLI in a workflow on `pull_request_target: closed` instead, or remove the markers so the
README does not display an empty section.

### MNT-15 (P2) A `recipe-health` issue does not name the file to edit, and assumes a binary nobody can install

**Where:** `.github/workflows/recipe-health.yml:109-131` (the body), `:101-107` (dedup), `:146-162` (auto-close),
`:135` (the weekly comment).

**What:** Judged against the standard the task sets - actionable by a newcomer who has never seen the codebase.
What the body gets right: it names the recipe, dates the failure, explains that the cause is almost always
upstream, links `CONTRIBUTING.md`, gives the last 40 log lines, and links the run. The dedup-by-title-prefix
logic is correct (the title carries the first-broken date, so an exact-title match would open a new issue
weekly), and the auto-close at 146-162 is the right half of the bargain. It carries `help wanted`,
`recipe-broken` and `recipe`, all three of which `labels.sh` creates.

Four gaps:

1. **It does not name the file to edit.** A newcomer is told a recipe is broken and given a log; the sentence
   "the fix is usually one line" is not followed by which line, or even which file. Adding "The files are
   `recipes/<name>/recipe.yaml` and `recipes/<name>/compose.yaml`" costs one line and removes the first
   question.
2. **The reproduction command cannot be run.** Line 120 gives
   `restored recipe test ./recipes/<name> --timeout 15m --pull always`, with a bare `restored`. There are no
   releases and no `install.sh`, so the only way to get the binary is
   `git clone && go build -o bin/restored ./cmd/restored`, and the command as written will not resolve. It also
   does not mention that Docker and restic must be installed.
3. **The excerpt is 40 lines of `--log-level debug`,** which per MNT-03 is docker and restic command lines. The
   most likely last 40 lines of a failing run are teardown commands, not the failure.
4. **It grows without bound.** Each Monday a recipe stays broken adds another 40-line comment to the same issue.
   Fifty-two comments a year saying the same thing is how a `help wanted` issue stops looking like an
   invitation. It is not spam in the "many issues" sense - the dedup works - but it is spam in the "nobody
   reads this thread" sense.

**Scenario:** Kim finds a `help wanted` `recipe-broken` issue through GitHub's global search, wants to make it
their first open-source contribution, reads the body, and has to open `CONTRIBUTING.md` to learn how to get a
binary and then guess which of two YAML files to change.

**Proposed fix:** Add to the body the two file paths, a two-line "get the tool" block matching
`CONTRIBUTING.md:23-27`, and the Docker/restic prerequisite. Comment only when the excerpt differs from the last
one, or at most monthly - `gh issue comment` is cheap to guard with a `date` check.

### MNT-16 (P3) The README contradicts itself on the recipe count

**Where:** `README.md:305-307` ("**v0.1, "it boots".** ... six bundled recipes") vs `README.md:222-227` (the
generated table, five rows) vs `README.md:16` ("five recipes ship") / `DECISIONS.md ADR-033` (six is the gate) /
`recipes/README.md:21` ("5 recipes.").

**What:** The roadmap line describes v0.1 as shipping six recipes; the table generated three sections above it
lists five; the pre-release note at line 16 says five. All three are correct about different things (the gate,
the present, the present), but a reader encountering them in order sees the project miscount itself. On the page
whose job is to establish that this tool does not manufacture confidence, an arithmetic inconsistency is a worse
signal than it would be anywhere else.

**Scenario:** A reader evaluating whether to trust the verdicts notices the count is wrong and applies that
judgement to everything else on the page.

**Proposed fix:** One word. `README.md:306` becomes "six bundled recipes (five ship today; see PROGRESS.md)", or
the roadmap bullet drops the number.

### MNT-17 (P3) `CHANGELOG.md` is required by the release checklist and does not exist

**Where:** `SPEC.md 2586` (release checklist item 1: "`CHANGELOG.md` updated, highlights written by a human") /
repository root (absent) / `PROGRESS.md:799-802` (listed under "Then", after the launch).

**What:** The release checklist's first item names a file that does not exist, and the plan puts creating it
after the launch. For a project measured in contributors, the changelog is also the place a contributor's name
appears in a release - the only such place, given MNT-14. Ordering it after the first release means the first
cohort of contributors is invisible in the artifact they helped ship. Tagging is a stop point, so discovering
the gap at tag time costs a human round trip that creating the file now avoids.

**Proposed fix:** Create `CHANGELOG.md` with an `## Unreleased` section before the first recipe PR is merged,
and add a line to `CONTRIBUTING.md` saying a recipe needs no changelog entry but a behaviour change does.

### MNT-18 (P3) The sequential fallback in `recipes.yml` drops the debug log and merges all verdicts

**Where:** `.github/workflows/recipes.yml:162-196` (compare line 184 with line 137, and line 189-196 with
144-151).

**What:** The more-than-twenty-recipe path (`test-sequential`) omits `--log-level debug`, which the matrix path
passes at line 137, and uploads a single `recipe-test-sequential` artifact covering every recipe rather than one
per recipe. It also carries `timeout-minutes: 350`. The path triggers on any change to `internal/`, `schema/`,
`tools/`, `docs/hints.yaml` or `go.mod` once there are more than 20 recipes - which is a maintainer's own pull
request, and exactly the run where the most debugging information is wanted. Today, with 5 recipes, it never
fires; it will fire on the day the project is at its most successful.

**Scenario:** At 24 recipes the maintainer changes `internal/check`. Three recipes fail. The one artifact is a
concatenated log with no per-recipe separation and no debug output, so the maintainer reruns all three locally.

**Proposed fix:** Add `--log-level debug` to line 184 for parity with line 137, and split the artifact by
writing `--report recipe-test-$name.json` per recipe in a loop. Both are two-line changes and both are much
cheaper now than the first time the path runs.

## Review burden per recipe PR

What a human still has to check on a `recipes/<name>/**` pull request after `recipes.yml` is green. Items marked
**(mechanical)** could be checked by a machine today and are not; those are the ones that stop this scaling.
Items marked **(judgement)** genuinely need a person, and are the honest limit of the bet in `SPEC.md 7.1`.

1. **`compose.yaml` has no top-level `volumes:` with `driver_opts`, and no `configs:` or `secrets:` block.**
   (mechanical - MNT-01). The highest-consequence item on this list: it is a host filesystem mount that
   `validate --strict` currently prints `ok` for, and the merged recipe is embedded into the shipped binary.
2. **The stage A verdict is not PASS-BY-STARTUP-REFUSAL; if it is, read the checks and confirm at least one
   depends on restored rows.** (mechanical - MNT-05). Stage A executed zero checks in that case, so the green
   badge is evidence about the application's boot behaviour, not about the recipe.
3. **`default_path` for every input is plausible for a real deployment.** (judgement). Nothing checks it: the
   harness builds its own tree and exports into it, so a recipe with `default_path: /srv/wrong/place`
   round-trips perfectly and then fails for every user. This is the largest true gap in the bet, and it is
   irreducible - it needs somebody who runs the application.
4. **The recipe README's "which of my directories is this input" section is accurate.** (judgement). Same reason
   as 3, and it is the section a user actually reads.
5. **`test.seed` goes through the application's front door, or says in a comment that it does not.**
   (judgement). `SPEC.md 7.4` and `CONTRIBUTING.md:96-101` both require the comment; nothing enforces it. A
   recipe that seeds its database directly and asserts on the same rows proves the restore path but not the
   application path, and only the comment records that.
6. **The data-sensitive check is not passing on rows the application creates for itself.** (judgement, mostly
   mechanical). Stage A catches this when the application boots empty - the Paperless `auth_user` trap is
   exactly this, and stage A does catch it. It does not catch it under PASS-BY-STARTUP-REFUSAL, so this
   collapses into item 2.
7. **`metadata.maintainers` is a real GitHub handle, not `@your-handle` or `@TODO`.** (mechanical - MNT-10,
   MNT-13). `--strict` rejects only an *empty* list.
8. **No placeholder survives from the scaffold: `todo_replace_this_table`, `example/template`, a `TODO:` left in
   a title or description.** (mechanical - MNT-13).
9. **Every image is pinned to a tag that exists, and ideally a digest.** (partly mechanical).
   `schema/compose-safety.schema.json:34` and `safety.go:251-257` reject `latest` and an untagged image; nothing
   checks that the tag resolves, though `recipe test` pulling it is a de facto check.
10. **The round trip fits the budget with room.** (mechanical). The PR template asks for the time as free text
    and `recipes.yml` enforces `--timeout 15m`; reading the actual duration from the run tells you whether the
    recipe sits at 2 minutes or at 14.
11. **The generated index was regenerated.** (mechanical - MNT-02). Today this shows up as a red `ci /
    generated` that `CONTRIBUTING.md` told the contributor to ignore.
12. **The failure, if any, was explained to the contributor.** (unavoidable today - MNT-03). Because the harness
    discards the inner report, every failing recipe PR currently requires the maintainer to reproduce locally
    and translate. This is the item that turns 10 PRs a week into a part-time job, and it has the best
    effort-to-payoff ratio to fix.
13. **The PR is not a duplicate of an in-flight one for the same application.** (mechanical - MNT-11). No
    claiming convention exists, and 50 `good first issue` tickets are about to be opened at once.

Items 1, 2, 7, 8, 11 and 13 are all mechanical and all currently manual. Closing them leaves 3, 4 and 5 - three
judgement calls, all of the same shape ("does this describe the real application?"), all readable in about two
minutes from the recipe README. That is a merge a maintainer can do ten times a week. Today's list is not.
