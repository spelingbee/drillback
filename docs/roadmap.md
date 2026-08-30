# Ideas parked here

Things that came up while building, that are deliberately not being built. An idea in
this file is cheaper than an idea in the code.

## From session 2

- **A `--dry-run` for `check`.** Resolve, validate, and print what would happen
  without starting anything. `recipe show --inputs-only` answers most of it already.
- **A size ratio warning.** The broken-backup demo is recognisable at a glance because
  the dump is 489 bytes next to a 102 KiB data directory. drillback has both numbers in
  the report and could say so without a hint rule firing.
- **`drillback doctor`.** Already on the v0.2 roadmap in SPEC.md; session 2 produced
  three environment failures (no docker daemon, no restic, no C toolchain for `-race`)
  that a doctor command would have named in one line.
- **Detecting an application that rebuilt its own schema.** When the dump carries
  nothing and the application migrates an empty database, the tables exist and are
  empty. drillback could compare the dump's size and statement count against what the
  application produced and say which of the two it is looking at.
- **A per-check `retries` key.** Deliberately not added: checks run once, on purpose,
  and a check that needs a retry is a ready probe wearing a disguise.

## From session 3

- **Seed a real document into Paperless-ngx.** The recipe proves the database and the
  media directory round-trip; it does not drive a PDF through consumption, because
  that needs either a multipart upload or a file dropped into the consume directory
  and then waited for, and the step vocabulary has neither. Two ways out: a `wait`
  step kind that retries until a condition holds, or a `file` step kind that copies a
  test asset into a service. Either would also help several other applications.
- **A `wait` step for the harness.** Steps run once by design, which is right for
  `create a user` and wrong for `now wait until the consumer has picked that up`. A
  ready probe already has the retry machinery; a step that reuses it would be small.
- **Multipart upload as a step or check kind.** The reason several document and photo
  applications cannot be seeded through their own front door.
- **A `mysql-dump` input kind.** `recipe init --compose` already recognises a MySQL or
  MariaDB service and says, in the file it writes, that drillback cannot restore it
  yet. That message is a promise to somebody.
- **A recipe-level `prepare` hook.** The Nextcloud recipe declares a compose service
  that chowns the restored tree and lays down a config overlay before the application
  starts. It works, and it is exactly what a person does by hand - but every recipe
  for an application that is fussy about ownership will now copy it, and a copied
  twenty-line service is a convention rather than a mechanism.
- **`drillback check --all` reading `drillback.yaml`.** Session 4's job, and the reason
  the nudge reads `defaults.nudge` out of that file through a deliberately narrow
  one-key reader rather than through `internal/config`, which does not exist yet.
- **A `smoke.yml` workflow.** SPEC.md section 11.3 specifies a fresh-clone smoke test
  and it is not written. The `unit` job proves `go test ./...` is green without docker
  or restic, which is most of what it was for.
- **Star counts in `docs/recipes-wanted.txt` go stale.** The file records the query and
  the date it was gathered, and says to re-run rather than edit. A tiny workflow could
  refresh it monthly; whether that is worth a scheduled job is not obvious.

## Explicitly not doing

Kubernetes, a hosted service, a GUI, restoring to production. See SPEC.md section 14.
