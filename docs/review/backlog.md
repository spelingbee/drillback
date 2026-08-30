# The P2 and P3 backlog

The session 4 brief said: fix every P0 and P1, and for each P2 and P3 either fix it or
open a GitHub issue labelled `help wanted`, because these are contributor entry points
and hoarding them is the opposite of what this project is for.

**The issues cannot be opened yet.** This repository has no remote and is not public;
making it public is stop point 4 in [CLAUDE.md](../../CLAUDE.md) and belongs to a human.
So this file is the issues, written out, with the command that files them. It is not a
substitute for filing them - it is the thing to run the hour after the repository goes
public, and before `scripts/recipes-wanted.sh`, so that a stranger arriving on day one
finds work to do.

```sh
# after the repository is public and scripts/labels.sh --apply has run
./scripts/backlog-issues.sh --dry-run     # read what it would file
./scripts/backlog-issues.sh --apply
```

Every issue body points at the finding in this directory, which carries the
reproduction. That is deliberate: a `help wanted` issue that only says what is wrong
costs the first contributor an hour of archaeology.

---

## Fixed in session 4, and not in the backlog

These were reported as P2 or P3 and are done. Listed so nobody files them again.

| finding | what was done |
|---|---|
| ARCH-12 | the harness carries and renders the inner check report (ADR-061) |
| SEC-07 | `recipe show` lists every image the recipe will pull, before anything else |
| SEC-10 | `install.sh` with checksum verification, SBOMs, the container image, `docs/docker.md` |
| UX-07, FC-12 | `--strict` prints `WARN`, names the recipe on each warning, and says why the exit code is 2 |
| UX-09 | the documented glob is `./recipes/*/`, which is the one that works |
| UX-13 | the README quick start's inputs agree with `recipe show` |
| MNT-10 | `recipe init --compose` no longer emits `:latest`, and the image error explains itself |
| MNT-16, FC-16 | the README says five bundled recipes in both places |
| MNT-17 | `CHANGELOG.md` exists |
| FC-06 | a recipe-only pull request no longer fails `ci / generated` (ADR-060) |

---

## The backlog

Grouped by what a contributor would need to know to pick one up.

### Good first issues - self-contained, no Docker needed

| id | title | why it is a good first issue |
|---|---|---|
| UX-15 | `--keep` prints a POSIX cleanup command and does not say what it kept | one function, obvious right answer, immediately visible |
| UX-17 | Report width is fixed at 78 columns and `COLUMNS` is ignored | one constant becomes a lookup, with a golden test already in place |
| UX-16 | Four flag combinations are accepted without complaint | four `if` statements and four error strings |
| UX-14 | JSON error reports carry `null` arrays and an empty `run` block | struct tags, and a golden file to update |
| ARCH-15 | Four discarded parameters and one dead struct field | pure deletion, and the linter agrees |
| ARCH-13 | Hint selection prioritises subject order over rule order, against SPEC 6.1 | a sort, and the hint tests are the best-covered in the tree |
| FC-11 | The `db/tables-empty` hint is written about Gitea and printed for every recipe | writing, not code; `docs/hints.yaml` is the easiest file to contribute to |
| MNT-16 | *(fixed)* | |

### Needs a little of the codebase

| id | title |
|---|---|
| UX-08 | `recipe init` tells the user to run a command that exits 2, and never mentions `recipe test` |
| UX-10 | Raw OS and Go plumbing errors reach the user in five places |
| UX-11 | `check --help` dropped the `Environment:` block and every `--input` example |
| UX-12 | The ASCII fallback is undocumented, unreachable by flag, and incomplete |
| FC-07 | `docs/recipe-spec.md` drops item-level constraints and object sub-fields, so a recipe written from the reference alone does not validate |
| FC-08 | The `--compose` scaffold loses the only documentation of `profiles: [test]` and `${RESTORED_TEST_ASSETS}` |
| FC-09 | The generated recipe README omits an input and names the database after the service |
| FC-10 | The contribution link never appears for a normally commented recipe, and its fallback prints a `cp` that is wrong on Windows |
| FC-13 | Every documented command goes through `make`, with no fallback for a host that does not have it |
| MNT-09 | Four of the ten labels are created and never applied by anything |
| MNT-11 | Fifty `good first issue` tickets, and no way to claim one |
| MNT-12 | CONTRIBUTING omits the licensing, commit-message and question answers a first-timer needs |
| MNT-13 | `recipe validate --strict` passes the TEMPLATE verbatim, so the placeholder loop costs a Docker cycle |
| MNT-14 | The all-contributors flow cannot run, and CONTRIBUTING does not mention it |
| MNT-15 | A `recipe-health` issue does not name the file to edit, and assumes a binary nobody can install |
| MNT-18 | The sequential fallback in `recipes.yml` drops the debug log and merges all verdicts - and with the registry at twenty recipes (2026-08-30), a pull request touching every recipe is one recipe away from triggering it |
| MNT-19 | `contributors.sh` reports zero contributors with exit 0 when the repository query fails - an API error reads as "0" on a dashboard. Found by the 2026-08-30 maintainer session, not the session 4 reviews |
| ARCH-06 | restic's command lines and stderr never reach the run's debug log |
| ARCH-14 | `internal/workspace` does not own three paths inside the workspace |
| ARCH-16 | The recipe format is described in four places |
| SEC-11 | The `vars` secret warning specified in SPEC.md 9.3 is not implemented |
| SEC-12 | Templates in `default_path` are silently not expanded, and the `..` guard runs pre-render |
| SEC-13 | Defence in depth: an unreachable `..` check, and argv passed to `docker compose exec` without `--` |
| FC-14 | Everything that points at `github.com/spelingbee/restored` 404s until the repository is public |
| FC-15 | An interrupted run left a Docker network behind |

### Security, and worth a maintainer rather than a first-timer

These four are `help wanted` too, but they change a trust boundary, so they want a
design decision written down before the code.

| id | title | the decision to make first |
|---|---|---|
| SEC-06 | `expect.glob` escapes the workspace and turns the report into a host filesystem oracle | is `glob` rooted at the input, or is it a path pattern that must be contained? |
| SEC-08 | The JSON report embeds 200 lines of every service's container log | how much log is worth the risk of somebody's data in a document people attach to bug reports? |
| SEC-09 | A `sql` check's `file:` is unconstrained and opens arbitrary host paths | should `file:` accept anything but a `{{ .inputs.<name>.path }}` template? |
| ARCH-09 | A killed run leaves a full copy of the user's backup in the temp directory, and nothing finds it | `restored doctor`, a startup sweep of stale workspaces, or both? |

### Architecture, for whoever picks up v0.2

| id | title | blocks |
|---|---|---|
| ARCH-11 | Every failure is an untyped string, which is the wall the notifiers will hit | notifiers |
| ARCH-08 | No test seam at the docker boundary: `probe`, `runner`, `compose`, `sqlite`, `dir` are all 0% | everything, slowly |
| ARCH-07 | Two lifecycle implementations that share code by copy | the harness and the runner drifting apart |
| ARCH-10 | `recipe test` grows the user's restic cache without bound | nothing, but it is somebody's disk |

---

## Why these were not fixed

Not because they do not matter. Because the brief asks for P0 and P1 fixed and the
rest *offered*, and because a project whose success metric is the number of distinct
external contributors with merged pull requests should not arrive at launch having
already done every job that is small enough for a stranger to finish.

The three that most tempt a maintainer to keep - UX-15, UX-17 and FC-11 - are the three
best first issues in the list, and they are deliberately left.
