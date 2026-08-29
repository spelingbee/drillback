# Progress

**Read this file first.** It is the handover between sessions: what exists, what is
next, what is stuck. Update it at the end of every session, and whenever a claim in it
stops being true.

Rules for this file:

- Every "tests pass" or "it works" claim must name the **exact command** and include the
  **tail of its actual output**. A claim without evidence is a guess, and a guess in a
  handover document is worse than a gap.
- Convert relative dates to absolute ones. "last week" is meaningless to the next
  session.
- Do not delete history. Append to the session log; move items between *Current state*,
  *Next steps* and *Blocked* rather than dropping them.

---

## Current state

**Phase:** specification complete, no code written.
**Version:** unreleased. No tags, no `go.mod`, no `cmd/`.
**Language of record:** English, everywhere. See [DECISIONS.md](DECISIONS.md) ADR-012.

What exists in the repository:

| Path | State |
|---|---|
| `SPEC.md` | Complete for v0.1. 14 sections: problem, CLI surface, recipe format + JSON Schema, run lifecycle, report format, hints, round-trip harness, nudge, threat model, testing, CI, release, layout, roadmap. |
| `DECISIONS.md` | 35 ADRs. 001–012 are the fixed decisions from the brief; 013–035 were made while writing the spec. |
| `docs/name-check.md` | Name research for `restored` and four fallbacks across GitHub, npm, PyPI, crates.io, Homebrew, Docker Hub and three TLDs, with a ranked recommendation. **Awaiting a human decision.** |
| `CLAUDE.md` | Conventions and stop-points for future sessions. |
| `PROGRESS.md` | This file. |
| `LICENSE` | Apache-2.0, copyright "The restored Authors". |
| `.gitignore`, `.editorconfig` | Written. |

What does **not** exist yet, and is not expected to:

- No Go code of any kind. No `go.mod`, no `cmd/restored`, no `internal/`.
- No `recipes/` directory. The Gitea and Uptime Kuma recipes exist only as examples
  inside SPEC.md § 3.1 and § 3.2.
- No `schema/*.json`. Both schemas exist only inside SPEC.md § 3.4 and § 3.5.
- No `docs/hints.yaml`. The 16-rule catalog exists only inside SPEC.md § 6.2 — see
  ADR-034; the implementation session must **extract** it, not rewrite it.
- No `.github/workflows/`. The CI plan is SPEC.md § 11.
- No `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`.

**Verification status:** nothing to verify. No build, no test suite, no command has been
run against this project, because there is nothing executable in it. The next session is
the first one that can make a "tests pass" claim, and it must do so with a command and
its output.

---

## Session log

### Session 1 — 2026-08-30 — Specification

**Goal:** produce the specification and the project's working documents. No application
code.

**Done:**

1. **Name check.** Queried GitHub (authenticated `gh search repos` + `gh api users/…`),
   npm, PyPI, crates.io, Homebrew formula and cask APIs, Docker Hub user and library
   endpoints, and RDAP/DNS for `.dev`, `.sh`, `.io`, across all five candidate names.
   Findings written to `docs/name-check.md` with a ranked recommendation. Nothing was
   renamed.

   Three findings drove the recommendation:
   - `restore-drill` is one hyphen from [`ahmadpiran/restoredrill`](https://github.com/ahmadpiran/restoredrill)
     — 87 stars, Go, MIT, actively maintained, and doing very nearly the same thing for
     PostgreSQL. Rejected.
   - `restored` is legally and technically usable — no exact repo collision above 100
     stars, and the Homebrew formula name is free — but the GitHub user, the Docker Hub
     user, the npm package, `restored.dev` and `restored.io` are all taken, and the word
     reads as a Unix daemon (`restore` + `d`; `restored` is a real Apple iOS daemon).
     Usable, with a discoverability cost.
   - `drillback` is the only candidate clean on every registry and all three TLDs.

2. **`SPEC.md`.** Written in full: exact `--help` output for every command; a
   `restored.yaml` with four targets and two restic sources plus a cron line; complete
   `recipe.yaml` + `compose.yaml` for Gitea + PostgreSQL and for Uptime Kuma (SQLite),
   with seed SQL; the recipe JSON Schema and a separate compose-safety schema encoding
   the isolation rules; the run lifecycle as an eight-state machine with a per-state
   budget and exit-code table; TTY report mocks for PASS and RESTORE UNUSABLE, both
   labelled as mocks that must never be copied into the README; the JSON report with a
   stability contract; a 16-rule `hints.yaml`; the round-trip harness including the
   throwaway restic setup, the 20-minute budget and the CI recipe-selection shell; the
   nudge with its 6,000-character fallback; the threat model split into mitigated,
   accepted, and explicit non-mitigations; the testing pyramid; five CI workflows with
   runtime budgets; the release process; the repository tree with package-boundary
   rules; and the v0.2/v0.3 roadmap.

3. **`DECISIONS.md`.** 35 ADRs. The 12 fixed decisions recorded verbatim in intent, plus
   23 decisions made while writing the spec — every point where the brief left a choice.

4. **`PROGRESS.md`** and **`CLAUDE.md`** written.

5. `git init`, `LICENSE` (Apache-2.0, fetched from apache.org and the appendix copyright
   line filled in), `.gitignore`, `.editorconfig`.

**Not done, deliberately:** no application code, no `go.mod`, no `recipes/`, no
`schema/`, no workflows. Session 1's constraint.

**Evidence.** There is no code, so there is no build or test claim. The documents
themselves were machine-checked, and the checks found and fixed one real bug.

Every fenced JSON and YAML block in SPEC.md was parsed; both schemas were checked as
draft 2020-12; both example recipes were validated against `recipe.schema.json` and both
example compose files against `compose-safety.schema.json`; and twelve synthetic
dangerous compose files were checked to confirm the safety schema rejects them.

```text
$ python - <<'EOF'   # extract the fenced blocks from SPEC.md and validate them
[schema] recipe.schema: valid draft 2020-12
[schema] compose-safety.schema: valid draft 2020-12
[valid]  gitea/recipe.yaml
[valid]  uptime-kuma/recipe.yaml
[valid]  gitea/compose.yaml
[valid]  uptime-kuma/compose.yaml

negative controls (each MUST be rejected):
  rejected   privileged
  rejected   ports
  rejected   network_mode host
  rejected   pid host
  rejected   host bind mount
  rejected   long bind mount
  rejected   non-internal net
  rejected   image :latest
  rejected   cap_add SYS_ADMIN
  rejected   build
  rejected   devices
  rejected   seccomp unconf

all dangerous constructs rejected: True
```

**Bug found and fixed by this check:** the first draft of `$defs/internalURL` rejected
every example recipe, because its pattern allowed only a numeric port and every recipe
writes `http://gitea:{{ .vars.gitea_port }}/`. The fix was to permit `{{ ... }}` in the
port and path, and it exposed a design point that was implicit and is now written down
as SPEC.md § 3.4.1: the **recipe** schema validates the file *as written*, with
templates unexpanded, so that editors and the independent-validator CI job can use it,
while the **compose-safety** schema validates *after* interpolation, because its whole
purpose is to inspect resolved values.

The 16 hint rules were also checked: unique ids, all four required fields present, every
`match` compiles as a regular expression, and the error string in the RESTORE UNUSABLE
mock (SPEC.md § 5.1) is matched first by `postgres/relation-missing` — the rule that
mock claims fires. Rule ordering and the mock agree.

```text
hint rules: 16
all rules have id/match/title/text and compile: True
first rule matching the RESTORE-UNUSABLE mock error: postgres/relation-missing
spec claims: postgres/relation-missing -> MATCH
```

Note for session 2: this validation was ad-hoc Python. It must become the `schema` CI
job (SPEC.md § 11.1) and the hint fixture tests (§ 10.1), against the extracted files
rather than against fenced blocks in a Markdown document.

Other commands run this session were read-only research (`gh search repos`, `gh api`,
`curl` against public registry APIs, `nslookup`) plus `git init` and file writes.

---

## Next steps

In order. Each is sized to be finishable and committable on its own.

### Immediately blocked on a human — see *Blocked*

0. **Decide the name.** Everything below embeds it.

### Session 2 — scaffold, once the name is decided

1. `go mod init github.com/<owner>/<name>`, pinned to the current stable Go toolchain.
2. `cmd/<name>/main.go` with nothing but wiring, and `internal/cli` with the cobra
   command tree — every command and flag from SPEC.md § 2, each returning
   "not implemented" with exit 2. The `--help` output is the acceptance criterion: it
   must match SPEC.md § 2 exactly. Diff it and fix the spec if reality is better.
3. `schema/recipe.schema.json` and `schema/compose-safety.schema.json`, extracted
   verbatim from SPEC.md § 3.4 and § 3.5.
4. `docs/hints.yaml`, extracted verbatim from SPEC.md § 6.2 (ADR-034).
5. `recipes/gitea/` and `recipes/uptime-kuma/`, extracted verbatim from SPEC.md § 3.1
   and § 3.2.
6. `.github/workflows/ci.yml` with the `lint` and `unit` jobs only.
7. `Makefile`: `build`, `test`, `lint`, `test-integration`, `demo`.
8. `.gitattributes` with `* text=auto eol=lf`. Session 1 was created on Windows, where
   git's default `core.autocrlf=true` would have rewritten every file to CRLF and
   contradicted `.editorconfig`'s `end_of_line = lf`. It was worked around with a
   repository-local `git config core.autocrlf false`, which is **not** committed and does
   not travel — the next Windows contributor gets the wrong endings. `.gitattributes`
   was outside session 1's allowed file list, so it is the first thing session 2 adds.

Commit after each numbered item. Green `go build ./...` and `go test ./...` before every
commit, with the command and output recorded here.

### Session 3 — validation first

`internal/recipe` and `internal/recipe/safety`, with `restored recipe validate` and
`restored recipe show` working end to end. This is the right first vertical slice: it is
pure, it is fully unit-testable with no Docker, and it makes the two extracted recipes
prove themselves immediately. Table-driven tests, one row per schema constraint
(SPEC.md § 10.1).

### Session 4 — the happy path

`workspace` → `source/dir` → `compose` → `probe` → `check` → `report`. Deliberately use
`--source dir` first and leave restic for session 5, so the first working `check` needs
no backup repository. Target: `restored check --recipe ./recipes/uptime-kuma --source
dir --from ./testdata/uk-tree` prints a real PASS.

### Session 5 — restic, then the harness

`source/restic` with recorded-JSON unit tests for snapshot selection, then
`internal/harness` for `recipe test`, then `recipes.yml` in CI.

### Then

`hints`, `nudge`, the remaining four recipes (ADR-033), goreleaser, the ghcr image,
`install.sh`, and the fresh-clone smoke test.

---

## Blocked

| # | Item | Since | Blocking | Needed from |
|---|---|---|---|---|
| B-1 | **The project name, and the GitHub owner.** `docs/name-check.md` recommends `drillback` (only fully-clean namespace); `restored` is usable at a discoverability cost. The owner determines the `go.mod` module path, the schema `$id`s, the nudge URL, the ghcr image, and the Homebrew tap. All documents currently use the literal placeholder `OWNER` (ADR-035). | 2026-08-30 | `go mod init`, and therefore all code | A human decision. Nothing else is required — the research is complete. |

Nothing else is blocked. B-1 blocks session 2 entirely; there is no useful code that can
be written around it, and inventing a module path to unblock ourselves would mean a
rewrite of every import in the repository.

---

## Open questions for a human

Not blocking, but cheaper to answer now than after v0.1 ships.

1. **ADR-023 — is a failed ready probe exit 1 or exit 2?** The spec says 1: an app that
   will not start from its restored data is an unusable restore. The cost is false alarms
   when a recipe's pinned image disappears upstream or the host is slow. This is the
   single most reversible-now, expensive-later decision in the specification.
2. **ADR-030 — should digest pinning be required rather than encouraged?** Requiring it
   is strictly safer and makes recipes go stale in weeks, which would damage the one
   metric this project has agreed to care about.
3. **ADR-033 — are six recipes the right gate for v0.1?** Six stresses the format
   properly and delays the release. Three would ship sooner and freeze `apiVersion` on
   thinner evidence.
4. **Is Nextcloud actually achievable inside the isolation rules?** It is on the v0.1
   recipe list specifically because it is awkward, and if it cannot be done the rules
   need a documented exception rather than a quiet one.
