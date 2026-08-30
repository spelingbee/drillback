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
- Nothing has been published, tagged, pushed, posted or filed.

---

## 1. Repository settings

None of these can be done from a shell, and three of them are load-bearing for
documents that are already written.

| # | Setting | Where | Why it cannot wait |
|---|---|---|---|
| 1.1 | **Private vulnerability reporting: on** | Settings > Code security | `SECURITY.md` and `CODE_OF_CONDUCT.md` both route here. Off by default; if it is off when the repository goes public, the only advertised security channel and the only advertised conduct channel both 404 (MNT-07). |
| 1.2 | **Discussions: on** | Settings > General > Features | It is the first entry in the issue chooser and the channel `CONTRIBUTING.md` now sends questions to. Off by default (MNT-08). |
| 1.3 | **Issue templates** | already in `.github/ISSUE_TEMPLATE/` | Nothing to enable; confirm the chooser renders after 1.2. |
| 1.4 | **Actions: require approval for first-time contributors** | Settings > Actions > General | Leave this **on**. `recipes.yml` runs container images a pull request names. `CONTRIBUTING.md` now warns contributors that their first run waits, so the setting and the documentation agree (MNT-06). |
| 1.5 | **Branch protection on `main`** | Settings > Rules | Required checks, exactly these names: `ci / lint`, `ci / generated`, `ci / unit`, `recipes / verdict`. **Not** `recipes / test (<name>)` - those are matrix jobs whose names depend on which recipes a pull request touched, and they are skipped entirely when a change touches none. `recipes / verdict` is the aggregate that always runs. `ci / integration` is real and green, but it needs Docker and is the slowest job here; require it only if you are willing to wait for it on every merge. |
| 1.6 | **Allow `refresh-registry.yml` to push to `main`** | Settings > Actions > General > Workflow permissions | It needs read and write. If branch protection blocks it, add the GitHub Actions bot as an exception - otherwise the recipe tables silently stop updating and the next contributor gets the red check ADR-060 removed. |

---

## 2. Labels, before anything applies one

```sh
./scripts/labels.sh                 # dry run: read what it would create
./scripts/labels.sh --apply
```

`gh` will fail on any workflow or template that references a label that does not exist,
so this comes before the repository is public and before any issue is filed.

---

## 3. Make the repository public

**Stop point 4.** After this, everything in the repository is quotable and every link
in it resolves. `docs/name-check.md` still recommends `drillback` over `restored`, and
ADR-036 records that the rename is one `grep -rl spelingbee/restored | xargs sed -i`
plus a `go mod edit` - *until something is published*. This step is the last cheap
moment to change the name.

---

## 4. Tag and release

**Stop point 1.** Work through SPEC.md 12.6 first, then:

```sh
git tag -a v0.1.0 -m "restored v0.1.0"
git push origin v0.1.0
```

`release.yml` then runs the full suite, builds with goreleaser, and creates a **draft**
release. It publishes nothing: `.goreleaser.yaml` sets `release: draft: true`.

Before publishing the draft, and this is checklist item 12.6.4:

```sh
# download the draft's own linux asset and install it with the script users will use
./install.sh --version v0.1.0
restored version
restored recipe validate ./recipes/*/ --strict
```

Then publish the draft in the GitHub UI. **Stop point 6** - that is the moment
`restored` becomes downloadable.

---

## 5. The container image

**Stop point 6.** Nothing pushes an image today: there is no workflow step that does,
on purpose. To publish one:

```sh
docker build -t ghcr.io/spelingbee/restored:0.1.0 -t ghcr.io/spelingbee/restored:latest \
  --build-arg VERSION=0.1.0 --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
echo "$GITHUB_TOKEN" | docker login ghcr.io -u spelingbee --password-stdin
docker push ghcr.io/spelingbee/restored:0.1.0
docker push ghcr.io/spelingbee/restored:latest
```

Then, in the GitHub UI:

- **Package visibility: public.** Packages > restored > Package settings > Change
  visibility. A package under a public repository is **private by default**, and
  `docs/docker.md` tells people to `docker run ghcr.io/spelingbee/restored:0.1.0`. If
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
5. re-run the release, or push the cask by hand from `dist/homebrew/Casks/restored.rb`;
6. verify: `brew install spelingbee/tap/restored && restored version`.

Until step 4, tagging writes the cask into `dist/` and pushes nothing. That is
deliberate: a tag should never be the thing that publishes to a package manager by
surprise.

---

## 7. Seed the work a stranger can pick up

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
