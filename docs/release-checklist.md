# Release checklist for v0.1.0

Everything here is a human's. Session 4 stopped at the line marked **STOP**; nothing
below it has been done, and nothing below it should be done by a session on its own
initiative. The stop points are in [CLAUDE.md](../CLAUDE.md); the ones this list
touches are 1 (tagging), 3 (posting), 4 (making the repository public, collaborators,
webhooks, deploy keys, Actions secrets) and 6 (publishing to a package registry).

The order matters. Settings before publishing, publishing before announcing, and the
labels before anything that applies one.

---

## 0. Before any of it: what is already true

- `go test ./...`, `go vet`, `golangci-lint`, `gofmt` and the English-only check are
  green on this commit. The evidence is in [PROGRESS.md](../PROGRESS.md), with the
  commands and the tails of their real output.
- `goreleaser check` passes and `goreleaser release --snapshot --clean` produces six
  binaries, six archives, six SBOMs, a checksum file and a Homebrew cask.
- The Linux binary out of that archive was run through `scripts/demo.sh` against real
  Gitea and PostgreSQL stacks and reached `PASS 5/5`, exit 0.
- Nothing has been tagged, published to a registry, or posted. The repository is
  public, the labels exist and the issues are filed (steps 2, 3 and 7 below).

---

## 1. Repository settings

Mostly done on 2026-08-31, on the private repository, via the API and verified in the
UI. Two settings turned out to be public-repository-only - GitHub refuses them on a
private one - so they moved to the go-public moment and are listed under 3.

| # | Setting | State on 2026-08-31 |
|---|---|---|
| 1.1 | **Private vulnerability reporting: on** | **Done on 2026-09-03**, by the human, once the repository was public; the Advanced Security page shows *Disable* for it. Dependency graph, Dependabot alerts, security updates and grouped security updates were switched on at the same time (`dependabot_security_updates: enabled` via the API). `SECURITY.md` and `CODE_OF_CONDUCT.md` both route here (MNT-07). |
| 1.2 | **Discussions: on** | **Done** (`gh repo edit --enable-discussions`), verified in Settings > Features. |
| 1.3 | **Issue templates** | **Done**: the chooser renders all four templates plus the three `config.yml` contact links, verified at `/issues/new/choose`. |
| 1.4 | **Actions: require approval for first-time contributors** | **Done, verified on 2026-09-03** via the API: `actions/permissions/fork-pr-contributor-approval` answers `first_time_contributors` (MNT-06). |
| 1.5 | **Branch protection on `main`** | **Done**, with the names the checks actually have - the `ci / ` prefix was a UI rendering, and `unit` is a matrix: required checks are `lint`, `generated`, `unit (ubuntu-latest)`, `unit (macos-latest)`, `unit (windows-latest)`, `verdict`, each pinned to the GitHub Actions app (id 15368). `integration` is deliberately not required: it is real and green, but it is the slowest job. Not `test (<name>)` - the recipes matrix is skipped entirely when a change touches no recipe; `verdict` is the aggregate that always runs. |
| 1.6 | **Allow `refresh-registry.yml` to push to `main`** | **Done**: workflow permissions are read and write (verified in the UI). One caveat now written down instead of discovered later: classic branch protection applies required checks to the Actions bot's direct pushes too, so `refresh-registry.yml`'s push may be rejected on its first real run. If it is, convert the protection to a ruleset with the GitHub Actions app as a bypass actor - the "exception" this row always meant - and retire this caveat. |

---

## 2. Labels, before anything applies one

```sh
./scripts/labels.sh                 # dry run: read what it would create
./scripts/labels.sh --apply
```

`gh` will fail on any workflow or template that references a label that does not exist,
so this comes before the repository is public and before any issue is filed.

---

## 3. Make the repository public - done

**Done by the human** before 2026-09-03 (session 9 found `isPrivate: false` on
arrival), and the two settings below were switched on and verified the same day; the
table in step 1 has the evidence. Kept here as written, because the order still
matters for anyone doing this again.

**Stop point 4.** After this, everything in the repository is quotable and every link
in it resolves. The name question is closed: the human chose `drillback` - the
`docs/name-check.md` recommendation - and ADR-070 records the executed rename. There
is nothing left to decide here; going public is now only about stop point 4 itself.

The moment the repository is public, two settings that GitHub refuses on a private
one (see step 1) become available and must be done immediately, before anything is
announced:

- **Private vulnerability reporting: on** (Settings > Code security) - the channel
  `SECURITY.md` and `CODE_OF_CONDUCT.md` both advertise;
- **Actions: require approval for first-time contributors** - verify it is on
  (Settings > Actions > General); it is the default, and `recipes.yml` runs container
  images a pull request names.

---

## 4. Tag and release

**Stop point 1.** Work through SPEC.md 12.6 first, then:

```sh
git tag -a v0.1.0 -m "drillback v0.1.0"
git push origin v0.1.0
```

`release.yml` then runs the full suite, builds with goreleaser, and creates a **draft**
release. It publishes nothing: `.goreleaser.yaml` sets `release: draft: true`.

Before publishing the draft, and this is checklist item 12.6.4:

```sh
# download the draft's own linux asset and install it with the script users will use
./install.sh --version v0.1.0
drillback version
drillback recipe validate ./recipes/*/ --strict
```

Then publish the draft in the GitHub UI. **Stop point 6** - that is the moment
`drillback` becomes downloadable.

---

## 5. The container image

**Stop point 6.** Nothing pushes an image today: there is no workflow step that does,
on purpose. To publish one:

```sh
docker build -t ghcr.io/spelingbee/drillback:0.1.0 -t ghcr.io/spelingbee/drillback:latest \
  --build-arg VERSION=0.1.0 --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
echo "$GITHUB_TOKEN" | docker login ghcr.io -u spelingbee --password-stdin
docker push ghcr.io/spelingbee/drillback:0.1.0
docker push ghcr.io/spelingbee/drillback:latest
```

Then, in the GitHub UI:

- **Package visibility: public.** Packages > drillback > Package settings > Change
  visibility. A package under a public repository is **private by default**, and
  `docs/docker.md` tells people to `docker run ghcr.io/spelingbee/drillback:0.1.0`. If
  this is missed, every reader gets `denied` and concludes the project is broken.
- **Link the package to this repository**, so the package page shows the README and
  inherits the licence.

---

## 6. The Homebrew tap

**Stop points 4 and 6.** Full instructions in
[docs/homebrew-tap.md](homebrew-tap.md). In order:

1. create the public repository `homebrew-tap` under the same owner - the name is not
   optional, `brew` maps `spelingbee/tap` to `spelingbee/homebrew-tap`;
2. add a `Casks/` directory with a `.gitkeep`;
3. create a fine-grained token with contents read/write **on `homebrew-tap` only**, and
   add it here as the Actions secret `HOMEBREW_TAP_GITHUB_TOKEN`;
4. flip `homebrew_casks[0].skip_upload` from `true` to `false` in `.goreleaser.yaml`;
5. re-run the release, or push the cask by hand from `dist/homebrew/Casks/drillback.rb`;
6. verify: `brew install spelingbee/tap/drillback && drillback version`.

Until step 4, tagging writes the cask into `dist/` and pushes nothing. That is
deliberate: a tag should never be the thing that publishes to a package manager by
surprise.

---

## 7. Seed the work a stranger can pick up - done

**Done on 2026-08-31**: the 38 review findings are issues #4-#46 (`help wanted`),
the recipe requests are the 35 `recipes-wanted` issues up to #76, and `recipe-health`
filed #1-#3 itself. The commands stay here because both scripts are idempotent by
title and are how a later batch gets filed.

```sh
./scripts/backlog-issues.sh                  # dry run: 38 issues from the reviews
./scripts/backlog-issues.sh --apply --limit 8   # the first eight, then read them
./scripts/backlog-issues.sh --apply             # the rest

./scripts/recipes-wanted.sh --dry-run           # read all fifty first
./scripts/recipes-wanted.sh --apply --limit 5   # five, then look at the repository
./scripts/recipes-wanted.sh --apply             # the rest
```

Both are dry runs by default and idempotent by title. Do the backlog first: a
`recipes-wanted` issue asks somebody to write something, and a `help wanted` issue from
the reviews hands them something already diagnosed.

---

## 8. Announcing it

**Stop point 3, and the one that cannot be taken back.** The launch is a one-shot
resource. Nothing in this repository should decide how to spend it, and no session
should post anything anywhere - including replying to something about this project.

Before that conversation is worth having, items 1 through 7 should be done and the
repository should look like somewhere a stranger can land: a README with a working
install line, a GIF that plays, green checks, and issues that are ready to pick up.

---

## What is deliberately NOT on this list

- **Signing the binaries.** Neither cosign nor an Apple Developer ID. Both cost money
  (stop point 5) and the cask's post-install `xattr` hook handles the visible symptom
  on macOS. Checksums and SBOMs ship today; signatures are a v0.2 decision.
- **A `latest` tag on the release.** GitHub does that; the tap and `install.sh` both
  resolve it through the API.
- **Announcing on the recipe-request issues.** Those are for people who arrive, not a
  channel to push into.
