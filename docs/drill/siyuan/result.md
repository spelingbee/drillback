# SiYuan - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the workspace folder | all of `/siyuan/workspace`, 38 MiB | **PASS** (exit 0, 3 of 3 checks) | [result.txt](result.txt) |
| B - the notes | `workspace/data` only, 11 KiB | **PASS** (exit 0, 3 of 3 checks) | [result-data-only.txt](result-data-only.txt) |

Both work. The finding is what the 38 MiB is made of.

## The numbers

After creating one notebook and one document:

```text
38M    workspace
 11K   workspace/data      <- the notebooks and the document
512K   workspace/temp
 37M   workspace/conf
```

and inside `conf`:

```text
 16K   conf/conf.json      <- the settings, the access code, the API token
1.0K   conf/ca.crt
1.0K   conf/ca.key
4.0K   conf/cert.pem
1.0K   conf/key.pem
 37M   conf/appearance     <- themes, icon sets, fonts
```

**37 of the 38 megabytes are the bundled themes and icon sets**, which ship in the image
and are written into the workspace on first boot. The notes - the reason the workspace
exists - are 11 KiB of it.

For a person taking daily snapshots of the documented directory, that is 37 MiB of
image content per machine, deduplicated by restic after the first snapshot but copied,
hashed and stored on the way there every time. It is not wrong to back it up. It is
worth knowing it is there.

## Reading B, in full

```text
  inputs     workspace  /siyuan/workspace   7.0 KiB  ...

  CHECKS
  ok  serves-and-asks-for-the-code  The kernel serves, and still asks for the access code
  ok  notebook-restored             The notebook that was backed up is on disk
  ok  document-restored             The document's text is on disk

  PASS  3/3 checks
```

The verbatim reports are in [result.txt](result.txt) and
[result-data-only.txt](result-data-only.txt).

Restoring only `data/` gives a working, protected instance with the notes in it. Two
reasons, and the second one surprised this drill:

- SiYuan writes a default `conf/` on boot, including a fresh `appearance` tree from the
  image;
- **the access auth code is a launch flag, not a stored setting.** The drill expected
  reading B to produce an instance with somebody's notes and no password on the door. It
  does not: the code comes from `--accessAuthCode` on the command line, so an instance
  restored without its configuration is exactly as protected as the command that started
  it.

What reading B does lose is the settings themselves - appearance, editor preferences,
the API token, and the local certificate pair under `conf/`. Nobody's notes, and nothing
that cannot be set again.

## What the documentation would need to say

There is no backup section at all, so the useful addition is a short one:

> Back up the workspace folder. If you want a smaller backup, `workspace/data` holds
> your notebooks, documents and assets; `workspace/conf` is settings plus the bundled
> themes and icons, which SiYuan writes again from the image if they are missing.

## Not tested

- **The data repo** (`repo` snapshots and cloud sync). It is SiYuan's own backup
  mechanism, it is keyed by a passphrase, and it deserves its own drill. Nothing here is
  claimed about it.
- **Assets.** The seeded document has no attachment, so `data/assets` was empty. Both
  readings include it by construction.
- **Encrypted workspaces.** Not tested.

## Reproduced

Both verdicts are PASS, so the two-run rule for failures does not apply. The leg was run
twice end to end anyway, before and after a change to the recipe (2026-08-30 13:36 UTC
and 13:38 UTC), and `restored recipe test recipes/siyuan` passes both stages
independently.
