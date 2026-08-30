# File Browser - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - all three documented volumes | `/srv`, `/database`, `/config` | **PASS** (exit 0, 4 of 4 checks) | [result.txt](result.txt) |
| B - the files, without the database | `/srv` and `/config`; `/database` empty | **PARTIAL** (`RESTORE UNUSABLE`, exit 1, 3 of 4 checks) | [result-no-db.txt](result-no-db.txt) |

Keep all three volumes and File Browser comes back exactly as it was. The finding is
what the second reading costs, and how quiet it is.

## Reading B, in full

```text
  inputs     data      /srv            47 B  1 file
             database  /database        0 B  0 files
             config    /config        130 B  1 file

  CHECKS
  ok  serves-ui             The interface is served
  ok  canary-file-restored  The file under the files root came back
  X   share-in-database     The share that pointed at that file is still in the database
                              expect  exit_code: 0
                              got     exit_code: 1
  ok  settings-restored     settings.json is there and names the database

  RESTORE UNUSABLE  3/4 checks
```

The verbatim report is in [result-no-db.txt](result-no-db.txt).

Three of the four checks pass. The interface is served, `settings.json` is right, and
the file is exactly where it was. What is gone is everything that was in
`filebrowser.db`: the users, their permissions and scopes, and every share link.

## Why that is worse than it sounds

File Browser bootstraps a database when it does not find one, and it prints the new
admin password once, to the console:

```text
2026/08/30 12:19:25 User 'admin' initialized with randomly generated password: TJHWNwg0mJQPWOX6
```

The documentation's own warning is about exactly this:

> The automatically generated password for the user `admin` is only displayed once. If
> you fail to remember it, you will need to manually delete the database and start File
> Browser again.

So the restore that reading B produces is not "File Browser with some settings missing".
It is a File Browser whose accounts are gone, and whose new administrator password was
printed to a container log during a restore nobody was watching - on the day something
went wrong badly enough to need a restore. Every share link ever handed out is dead, and
every non-admin user's scope, permissions and password are gone with them.

The tool's own hint fires and describes the shape correctly:

> **The path exists in the snapshot but contains nothing.** The backup ran, the
> directory was included, and it was empty when it ran. The classic cause is backing up
> a bind-mount path while the application actually stores its data in a *named volume*
> mounted over the top of it.

which is, in fact, the most likely way a person arrives at reading B by accident rather
than by choice: three volumes, one of them bind-mounted somewhere the backup can see it,
two of them named volumes the backup never looks at.

## The recipe's `share-in-database` check

[`recipes/filebrowser`](../../../recipes/filebrowser) is the recipe that produced both
verdicts. `filebrowser.db` is a Bolt database, so there is no `PRAGMA integrity_check`
to lean on and no SQL to run against it. The check greps the database file for the path
of the shared file:

```sh
grep -qa drill-canary.txt /database/filebrowser.db
```

Crude, and it answers precisely the question that matters: is this the database that was
backed up, or one the application made for itself thirty seconds ago? A file that exists
under `/srv` proves nothing about the database, and a database that opens proves nothing
about whether it is *yours*.

## The thing a reader should take away

This application prints, on every start:

```text
NOTICE: File Browser is being wound down.
NOTICE: The project is archived on 2026-09-01
```

The drill ran on 2026-08-30. Thirty-six thousand people star a project, its
documentation never mentions backups, and it is archived in two days. Everyone still
running it is about to move their data somewhere else, and moving data is a restore.

## What documentation would have said

There is no one left to ask for it, so this is written for the reader rather than as a
draft issue:

> Back up all three volumes. `/srv` is your files, `/database` is your users, their
> permissions and every share link, and `/config` is the daemon settings. If you restore
> `/srv` alone, File Browser will start, your files will be there, and it will create a
> new administrator whose password is printed once to the log.

## Not tested

- **Multiple users and scopes.** The instance had the bootstrapped admin only. The
  drill did not create a second user, because on 2.63.23 the user-creation API rejects
  the request with `the current password is incorrect` and the CLI cannot open the
  database while the server holds the Bolt lock. What is demonstrated is the share row,
  which lives in the same database.
- **The s6 image.** Only the bare Alpine image was tested.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 12:19 UTC and 12:21 UTC).
