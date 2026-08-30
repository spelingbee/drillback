# changedetection.io - result

| Reading | What was backed up, and how it was restored | Verdict | Report |
|---|---|---|---|
| A - the Backups page's zip | `changedetection-backup-20260830130942.zip`, 117 KiB, unzipped into `/datastore` as the wiki page says | **PASS** (exit 0, 3 of 3 checks) | [result.txt](result.txt) |
| Control - the datastore | a copy of `/datastore`, put back and started | **PASS** (exit 0, 3 of 3 checks) | [result-datastore.txt](result-datastore.txt) |

The second application in this drill where the documented backup and the documented
restore both work as written.

## Reading A, in full

```text
  inputs     backups  /srv/changedetection-backups  117.0 KiB  1 file

  CHECKS
  ok  watch-list-renders        The watch list is served
  ok  watch-restored            The watch that was backed up is in the list
  ok  history-snapshot-on-disk  The watch's change history is on disk
                                  /datastore/<uuid>/history.txt

  PASS  3/3 checks
```

The verbatim report is in [result.txt](result.txt).

`history-snapshot-on-disk` is the check that earns its place. A watch is two things: a
row of configuration in `changedetection.json`, and a directory of snapshots of what the
page looked like. Restore the first without the second and changedetection.io comes up
looking correct, then reports the entire page as changed the next time it checks -
because as far as it knows it has never seen the page before. The zip contains both.

## What the wiki page gets right

> Important first is to stop the changedetection.io instance, as it will be monitoring
> and writing to the disk.

Two other applications in this drill lose data to exactly this - a copy taken while the
application was writing - and neither of them says it. It costs one sentence.

## Not tested

- **The Restore tab in the application.** The Backups page has one; the drill tested the
  command-line procedure from the wiki, which is the one a person restoring onto a new
  machine will reach for.
- **Password protection.** The instance had none, so the drill did not exercise a
  restore where the interface is locked and the password is in the datastore.
- **Watches with browser steps or attached notifications.** One watch, one snapshot.

## Reproduced

Both verdicts are PASS, so the two-run rule for failures does not apply. The leg was run
once end to end (2026-08-30 13:09 UTC), and `restored recipe test recipes/changedetection`
passes both stages independently.
