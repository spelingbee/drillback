# Mealie - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the data directory, container stopped | all of `/app/data` | **PASS** (exit 0, 5 of 5 checks) | [result.txt](result.txt) |
| B - the integrated backup zip | not tested; see below | - | - |

Mealie's backup page recommends one way in bold, and that way works.

## Reading A, in full

```text
  CHECKS
  ok  api-answers            The API answers with its version
  ok  db-integrity           The SQLite database passes PRAGMA integrity_check -> ok
  ok  users-present          The accounts survived the restore -> 1
  ok  recipes-present        The recipe that was backed up survived the restore -> 1
  ok  token-secret-present   The file that signs Mealie's sessions came back

  PASS  5/5 checks
```

The verbatim report is in [result.txt](result.txt).

Two of those checks are shaped by things this drill learned from other applications:

- **`recipes-present` names the recipe rather than counting rows.** A fresh Mealie
  creates its own administrator, so a check that counts users passes against a restore
  of nothing - the same trap Beszel walked into two legs earlier. A check that asks for
  `drill canary recipe` can only be satisfied by the backup.
- **`token-secret-present` looks for `.secret`**, which signs session tokens and sits
  beside the database rather than in it. Mealie writes a new one if it is missing, which
  logs everybody out; it is exactly the class of file that a restore of "just the
  database" loses without saying so.

## Reading B, and why it was not tested

The integrated backup writes a zip into `/app/data/backups` and is restored by uploading
it through the web portal, from an authenticated admin session, into a running instance.
That does not fit the shape of a `restored` recipe: the restore has to happen after the
application is up and before the checks run, and there is no ordering in a compose file
that expresses "run this once the service is answering, then let the checks start".

It is worth a drill of its own, and it is worth saying plainly that not testing it is a
gap in this result rather than a judgement about the feature.

What can be said from reading the page: the integrated backup is described as a backup
"of the database", and the tip immediately after it recommends the directory copy as
better - which is consistent with what the directory holds that the database does not
(`.secret`, `.session_secret`, and the recipe images under `recipes/` and `groups/`).

## Not tested

- The integrated backup and restore, as above.
- **PostgreSQL deployments.** The tip that names `/app/data` is explicitly about SQLite;
  with PostgreSQL the database is somewhere else and the directory is still needed for
  the images.
- **Recipe images.** The seeded recipe has no photo, so `recipes/<id>/` was empty. The
  directory copy covers it by construction; a database-only backup would not.

## Reproduced

The verdict is PASS, so the two-run rule for failures does not apply. The leg was run
once end to end (2026-08-30 13:22 UTC), and `restored recipe test recipes/mealie` passes
both stages independently.
