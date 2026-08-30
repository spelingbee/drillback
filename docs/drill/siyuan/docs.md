# SiYuan - the official backup documentation

- **Application:** [siyuan-note/siyuan](https://github.com/siyuan-note/siyuan), 46,036
  stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v3.8.2. Image `b3log/siyuan:v3.8.2`.
- **Documentation read:** 2026-08-30, from the repository README, which is where the
  Docker instructions and the FAQ live.

## Is there a backup page?

No. `grep -i backup` over the README finds the word only in the FAQ entry *What should I
do if the data repo key is lost?* and in the description of the `dejavu` component,
which is the data-repo snapshot library. There is no section that says what to copy.

## What the documentation does say

The Docker section describes one thing to mount:

> `--workspace`: Specifies the workspace folder path, mounted to the container via `-v`
> on the host

> `workspace_dir_host`: The workspace folder path on the host
> `workspace_dir_container`: The path of the workspace folder in the container, as
> specified in `--workspace`

> To simplify things, it is recommended to configure the workspace folder path to be
> consistent on the host and container, such as having both `workspace_dir_host` and
> `workspace_dir_container` configured as `/siyuan/workspace`.

and the FAQ describes the *data repo*, which is SiYuan's own snapshot and sync
mechanism, keyed by a passphrase:

> If the data repo key is correctly initialized on multiple devices previously, the key
> is the same on all devices and can be retrieved in Settings - Account & Sync - Local
> Data Repo - Data repo key - Copy key string

So the reading is the workspace folder. The data repo is a different thing - versioned
snapshots that can be pushed to cloud storage - and it is not what this drill tested.

## One thing worth recording, unrelated to backups

The README's Docker command is out of date for this version:

```text
Error: unknown flag: --accessAuthCode
```

v3.8.2 moved the server behind a `serve` subcommand, so the documented
`kernel --workspace=... --accessAuthCode=...` no longer starts. `kernel serve
--workspace=... --accessAuthCode=...` does. That is not a backup finding and it is not
written up as one, but it is the sort of thing a person restoring at two in the morning
would rather not discover then.

## The two readings

- **A: the workspace folder** - `/siyuan/workspace`, which is what the Docker section
  mounts.
- **B: the notes** - `workspace/data`, which is where the notebooks and documents are.
