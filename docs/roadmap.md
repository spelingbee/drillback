# Ideas parked here

Things that came up while building, that are deliberately not being built. An idea in
this file is cheaper than an idea in the code.

## From session 2

- **A `--dry-run` for `check`.** Resolve, validate, and print what would happen
  without starting anything. `recipe show --inputs-only` answers most of it already.
- **A size ratio warning.** The broken-backup demo is recognisable at a glance because
  the dump is 489 bytes next to a 102 KiB data directory. restored has both numbers in
  the report and could say so without a hint rule firing.
- **`restored doctor`.** Already on the v0.2 roadmap in SPEC.md; session 2 produced
  three environment failures (no docker daemon, no restic, no C toolchain for `-race`)
  that a doctor command would have named in one line.
- **Detecting an application that rebuilt its own schema.** When the dump carries
  nothing and the application migrates an empty database, the tables exist and are
  empty. restored could compare the dump's size and statement count against what the
  application produced and say which of the two it is looking at.
- **A per-check `retries` key.** Deliberately not added: checks run once, on purpose,
  and a check that needs a retry is a ready probe wearing a disguise.

## Explicitly not doing

Kubernetes, a hosted service, a GUI, restoring to production. See SPEC.md section 14.
