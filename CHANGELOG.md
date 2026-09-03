# Changelog

Notable changes to `drillback`. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The JSON reports carry their own contract: `schema_version` is `1` in both the check
report and the harness report, and within a major version fields are only ever added.

## [Unreleased]

Nothing yet.

## [0.1.0] - 2026-09-03

The first release. `drillback` restores a backup into a throwaway, isolated Docker
Compose stack, starts the application, and asserts that the data is actually there.

### Highlights

- **One command, one exit code.** `drillback check --recipe gitea --source restic
  --from /srv/backups` restores, boots, checks and cleans up in about a minute, and
  answers `PASS` (0), `RESTORE UNUSABLE` (1) or a tool error (2). A cron line is all
  the integration a NAS needs.
- **Twenty recipes, every one proved both ways.** Each ships in the binary and each
  has passed the round trip in CI: its checks must fail against an empty application
  and must pass against data that went out through a real restic backup and came
  back. That harness, not a reviewer's judgement, is what lets a stranger's recipe be
  merged.
- **Isolation is a schema, not a promise.** No privileged containers, no host
  namespaces, no published ports, no bind mount outside the run's own workspace, no
  Docker socket. A compose key the tool has not considered is rejected by name.
- **Failures come with a next step.** Eighteen hint rules turn `permission denied`,
  a schema-only dump, or a missing `config.php` into a sentence about what to fix.

### Added

- **`drillback check`** - the whole drill, end to end. Restores from a `restic`
  repository or from an already-restored tree, brings the stack up on an internal
  network with no published ports, loads any database dump, waits for the application
  to be ready, runs the recipe's checks, and tears everything down. `PASS` is exit 0,
  `RESTORE UNUSABLE` is exit 1, a tool error is exit 2.
- **`drillback.yaml`**, with `check --config`, `--target` and `--all`: sources,
  targets and defaults in one file, relative paths resolved against the file, an
  unknown key refused rather than ignored. `--all` runs every enabled target in file
  order, prints each report as it finishes, and exits with the worst outcome; a target
  that never ran is counted and forces exit 2 (ADR-067, ADR-068).
- **Twenty recipes**, each of which proves itself: `beszel`, `changedetection`,
  `convertx`, `filebrowser`, `freshrss`, `gitea`, `gogs`, `gotify`, `listmonk`,
  `mealie`, `memos`, `n8n`, `navidrome`, `nextcloud`, `open-webui`, `paperless-ngx`,
  `siyuan`, `trilium`, `uptime-kuma`, `vaultwarden`.
- **`drillback recipe test`** - the round-trip harness. Stage A runs a recipe's checks
  against an empty application and requires one to fail; stage B seeds real data
  through the application's own front door, backs it up with restic, destroys
  everything, restores, and requires every check to pass. This is what makes a
  stranger's recipe trustworthy without a maintainer understanding their application.
- **`drillback recipe init`**, with `--compose` to propose a recipe from a real
  `docker-compose.yml`: it finds the application, the database, the state directories
  and the port, writes a recipe that validates, and marks everything it could not
  decide as a `TODO` that names the decision.
- **`drillback recipe validate`** and **`drillback recipe show`**, with `--inputs-only`
  for the fastest answer to "which paths does this recipe want from my backup?".
- **Isolation enforced by a schema, not by discipline.** No privileged containers, no
  host namespaces, no published ports, no bind mount outside the run workspace, no
  Docker socket. The compose safety schema is an allow-list: a key drillback has not
  considered is rejected by name rather than granted silently.
- **A hint catalog** - 18 rules that turn a failure into a next step, extensible with
  `--hints FILE` and, deliberately, the easiest useful contribution to the project.
- **A JSON report** at `schema_version 1`, from `--json` or `--report FILE`, with the
  repository string scrubbed of any password.
- **`docs/drill/`** - a restore drill against fifteen applications' official backup
  documentation, quoted as written: the commands that were run, the reports the tool
  produced, and the root cause of every failure. `docs/drill/summary.md` has the
  totals and the patterns.
- **`install.sh`**, which detects the OS and architecture, verifies the release
  checksum, and refuses to run as root without `--system`.
- **A container image** bundling `drillback`, the Docker CLI, the Compose plugin and
  `restic`, for NAS users. `docs/docker.md` documents the exact invocation and is
  blunt about what mounting the Docker socket gives away.

### Security

Everything below was found by the independent reviews in `docs/review/` before the
first public release, and fixed before it. The full findings, with reproductions, are
in that directory.

- A recipe variable containing a line break could inject arbitrary YAML into the
  compose file that runs - `privileged: true`, `network_mode: host`, `pid: host` -
  past a safety schema that had already validated the file. Interpolation is now
  checked to change scalar values and nothing else about the document (ADR-056).
- A top-level named volume with `driver_opts: {type: none, device: /, o: bind}` was a
  bind mount of any host path, including the directory holding the Docker socket, and
  `recipe validate --strict` accepted it. The compose safety schema is now an
  allow-list at every level (ADR-057).
- A restic repository string carrying a password reached the report, the terminal and
  the debug log verbatim (ADR-059).
- `recipes.yml` interpolated a contributor-controlled directory name into a shell
  script, which is a textbook GitHub Actions injection.

### Fixed

- On Linux, an application image that re-owns its mounted data directory on startup
  (FreshRSS, Trilium, Nextcloud) made the restored files unreadable to `drillback`
  itself, and `load db` failed with `permission denied` before a single check ran.
  Every read of the workspace after the stack is up now goes through the Docker
  daemon, on every platform (ADR-071).
- A run that exceeded its `--timeout` was reported as `RESTORE UNUSABLE`, which
  accuses a backup that may be perfectly good. It is now a tool error, and the default
  budget is larger than the stage budgets inside it (ADR-058).
- Four of the seventeen hint rules could never fire, because hints were attached only
  on the paths that reached a verdict; and a tool error printed one line and no report
  at all, while `--report` still wrote the JSON.
- A failing round trip in CI reported which checks failed and discarded the query, the
  expectation, the observation, the service logs and the hint - all of which had
  already been computed (ADR-061).
- `recipe init --compose` dropped `depends_on` and `healthcheck`, and left the
  application's connection string pointing at credentials the generated file had
  stopped using, so the scaffolded stack could not come up.
- A recipe whose application refuses to start without data could pass the round trip
  with no check that counts anything, which is the false PASS this tool exists to
  destroy (ADR-064).
- An interrupted `--all` could exit 0 claiming a clean sweep when a signal landed
  during one target's teardown and the rest never ran.

### Known gaps

- `borg` and `kopia` sources are not implemented. `source.Source` is a real interface,
  so a third source is one `case` (ADR-063).
- A `mysql-dump` input kind. `recipe init --compose` recognises a MySQL service and
  says so in the file it writes.
- Binaries are not signed. Checksums and SBOMs ship with every release; signatures are
  a decision for a later version.
- 38 smaller findings from the pre-release reviews are open as `help wanted` issues
  on purpose: they are the places a first contribution fits.

[Unreleased]: https://github.com/spelingbee/drillback/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/spelingbee/drillback/releases/tag/v0.1.0
