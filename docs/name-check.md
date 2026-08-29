# Name check

**Date of check:** 2026-08-30
**Status:** research only — nothing has been renamed. The human decides.

Working name: `restored`. Fallbacks checked: `restore-drill`, `drillback`,
`bootproof`, `backupdrill`.

## Method

- **GitHub:** `gh search repos <name> --match name` (authenticated), then filtered
  to *exact* repository-name matches; plus `GET /users/<name>` to see whether the
  org/user namespace is free.
- **Package registries:** `registry.npmjs.org/<name>`, `pypi.org/pypi/<name>/json`,
  `crates.io/api/v1/crates/<name>`.
- **Homebrew:** `formulae.brew.sh/api/formula/<name>.json` and `.../cask/<name>.json`.
- **Docker Hub:** `hub.docker.com/v2/users/<name>/`, `.../repositories/<name>/<name>/`
  and `.../repositories/library/<name>/`.
- **ghcr.io:** not probed directly. GHCR is namespaced by the GitHub owner
  (`ghcr.io/<owner>/restored`), so its availability follows the GitHub-namespace
  row and is not an independent constraint.
- **Domains:** RDAP via `rdap.org`, cross-checked with authoritative NS lookups
  against 8.8.8.8. Controls showed `rdap.org` has **no RDAP coverage for `.io` or
  `.sh`** (`docker.io` and `dub.sh` both returned 404), so for those two TLDs the
  NS lookup is the primary signal and the result is "no delegation found", not a
  registry-confirmed availability. Treat `.io`/`.sh` rows as indicative.

## Summary matrix

Legend: **FREE** = available / no collision. **TAKEN** = occupied. **RISK** = available
but carries a confusion or discoverability problem.

| Check | `restored` | `restore-drill` | `drillback` | `bootproof` | `backupdrill` |
|---|---|---|---|---|---|
| GitHub repo, exact name, >100 stars | FREE | FREE | FREE | FREE | FREE |
| GitHub repo, exact name, any stars | RISK (8+ empty repos, 0 stars) | TAKEN (3 repos, top 1 star) | FREE | TAKEN (`bootproof/bootproof`, 6 stars) | FREE |
| Nearest high-star confusable | **`AnnotationsRestored` 193 stars** | **`ahmadpiran/restoredrill` 87 stars** | `DrillBack-to-LN` 0 stars | — | — |
| GitHub org/user namespace | **TAKEN** (User) | FREE | FREE | **TAKEN** (Org) | **TAKEN** (Org) |
| npm | **TAKEN** (0.1.0, dormant since 2022-05) | FREE | FREE | **TAKEN** (0.4.1, active 2026-07) | **TAKEN** (2.1.1, active 2026-08-29) |
| PyPI | FREE | FREE | FREE | FREE | FREE |
| crates.io | FREE | FREE | FREE | FREE | FREE |
| Homebrew formula | FREE | FREE | FREE | FREE | FREE |
| Homebrew cask | FREE | FREE | FREE | FREE | FREE |
| Docker Hub namespace | **TAKEN** (user, 0 public repos) | FREE | FREE | FREE | FREE |
| Docker Hub `library/` | FREE | FREE | FREE | FREE | FREE |
| `.dev` | **REGISTERED** (GoDaddy NS) | FREE | FREE | FREE | **REGISTERED** (Cloudflare NS) |
| `.sh` | free* | free* | free* | free* | free* |
| `.io` | **REGISTERED** (Namecheap NS) | free* | free* | free* | free* |

\* no NS delegation found; `.sh`/`.io` have no RDAP coverage via rdap.org, so this is
a DNS-level signal rather than a registry confirmation.

## Findings that matter

### 1. `restore-drill` collides with a live, direct competitor

[`ahmadpiran/restoredrill`](https://github.com/ahmadpiran/restoredrill) — 87 stars, **Go**,
MIT, created 2026-07-26, last push 2026-08-28, described as *"Proves your PostgreSQL
backups actually restore, before you find out the hard way."*

That is the same language, the same premise, the same implementation language, and it
is actively maintained. `restore-drill` vs `restoredrill` differs by one hyphen. This
is the single strongest finding in this document: **do not ship as `restore-drill`.**
It also means there is prior art worth reading before v0.1 — see *Competitive note*
below.

### 2. `restored` is unsearchable, and reads as a daemon

Three separate problems, none of them legal:

- **Grammatical collision.** "restored" is one of the most common English past
  participles, and it is the word every backup tool already prints in its own success
  message. `git restore` exists. Searching for `restored backup verify` will not
  surface this project; issue titles like "restored fails on Nextcloud" are ambiguous
  in the project's own tracker.
- **Daemon-suffix ambiguity.** Under Unix convention a trailing `d` means daemon —
  `sshd`, `dockerd`, `containerd`. `restored` parses as `restore` + `d`, and that
  reading is already occupied: [`JRHeaton/restored_pwn`](https://github.com/JRHeaton/restored_pwn)
  is *"an open source version of Apple's `restored_external` on the iPhone restore
  ramdisk"*. `restored` is a real iOS system daemon name. For a tool that is emphatically
  **not** a daemon (it is a one-shot CLI), this is an actively wrong signal.
- **Namespaces are chipped away.** The GitHub user `restored`, the Docker Hub user
  `restored`, the npm package `restored`, `restored.dev` and `restored.io` are all
  already occupied. None blocks the project — a GitHub org would have to be something
  like `restored-dev` (free) or a personal account, and `ghcr.io/<owner>/restored`
  is unaffected — but the "clean landing" version of this name does not exist. A
  Homebrew formula named `restored` **is** available, which is the one that would
  have hurt most.

### 3. `bootproof` and `backupdrill` are occupied by adjacent products

- `bootproof`: the GitHub org exists and ships *conceptually neighbouring* things —
  `bootproof/bootproof` ("zero-trust supervisor that boots"), `repo-proofer`,
  `receipt-gate` ("no merge without proof it runs"), and a `bootproof.github.io`
  calling itself "the proof company". npm `bootproof` was published 2026-07-03. An
  adjacent product on the same namespace is worse than an unrelated one: users will
  merge the two in their heads.
- `backupdrill`: org taken, `backupdrill/cli` is an active Supabase backup CLI
  (npm `backupdrill` published **2026-08-29**, i.e. the day before this check),
  `backupdrill.dev` registered. Same problem, closer domain.

### 4. `drillback` is the only candidate that is clean everywhere

Free on: GitHub repo name, GitHub org, npm, PyPI, crates.io, Homebrew formula and cask,
Docker Hub, and all three of `.dev` / `.sh` / `.io`. The only hits are two 0-star repos
using it in the unrelated BI/ERP sense of "drill back to the source record"
(`PrajwalMeti/DrillBack-to-LN`, `mwt2112-stack/P-Card-Drillback`).

Its weakness is semantics, not availability: "drillback" does not say *backup* or
*restore* to a self-hoster reading a one-line description, and it inherits a faint
business-intelligence connotation.

## Recommendation

Ranked, for a human to decide:

1. **`drillback`** — the only fully-clean namespace across every registry and TLD.
   Costs one sentence of explanation in the tagline; buys an unambiguous org, image,
   formula, domain and search result on day one. Recommended if discoverability and
   a clean `ghcr.io` / Homebrew story matter more than instant self-description.
2. **`restored`** — usable. Nothing legally or technically blocks it: no exact
   collision above 100 stars, no Homebrew formula, `library/restored` free, and GHCR
   follows whatever org is chosen. The whole cost is discoverability plus the daemon
   misread. Given the project's stated success metric is *external contributors with
   merged PRs*, and contributors have to be able to find the project first, this cost
   is not cosmetic. If chosen, take the org `restored-dev` and the domain `restored.sh`.
3. **A name not on this list.** Four of the five candidates have a namespace problem,
   which suggests the shortlist was drawn too close to the obvious vocabulary. Worth
   one more round before committing.
4. **`bootproof`**, **`backupdrill`** — not recommended; adjacent products already
   hold the namespace.
5. **`restore-drill`** — rejected; one hyphen away from an active 87-star Go project
   doing the same thing.

**Nothing in the repository has been renamed.** All documents in this session use the
working name `restored`. If the name changes, the strings to update are: the `go.mod`
module path, the `apiVersion: restored/v1` recipe field, the `RESTORED_INPUT_*` /
`RESTORED_VAR_*` environment variable prefixes, the compose project prefix
`restored-<runid>`, the config file name `restored.yaml`, and the nudge URL in
SPEC.md section *Contribution nudge*. They are listed here so the rename is one grep.

## Competitive note (not a naming question, but found during the check)

`ahmadpiran/restoredrill` (87 stars, Go, MIT, active) already occupies part of this
idea for PostgreSQL specifically. Before v0.1, someone should read it and write down
what `restored` does that it does not — the honest candidates are: docker compose
recipes rather than database-only checks, application-level "does the app boot and
serve" verification rather than "does the dump load", the round-trip self-validating
recipe harness, and restic snapshot integration. If that differentiation cannot be
stated in two sentences, the project needs rethinking more than it needs a name.

## Reproducing this check

```sh
for n in restored restore-drill drillback bootproof backupdrill; do
  gh search repos "$n" --match name --limit 60 \
    --json fullName,stargazersCount \
    --jq ".[] | select((.fullName|split(\"/\")[1]|ascii_downcase) == \"$n\")"
  gh api "users/$n" --jq .type
  curl -s -o /dev/null -w "npm %{http_code}\n"  "https://registry.npmjs.org/$n"
  curl -s -o /dev/null -w "pypi %{http_code}\n" "https://pypi.org/pypi/$n/json"
  curl -s -H 'User-Agent: name-check/1.0' "https://crates.io/api/v1/crates/$n"
  curl -s -o /dev/null -w "brew %{http_code}\n" "https://formulae.brew.sh/api/formula/$n.json"
  curl -s -o /dev/null -w "hub %{http_code}\n"  "https://hub.docker.com/v2/users/$n/"
  for t in dev sh io; do nslookup -type=NS "$n.$t" 8.8.8.8; done
done
```
