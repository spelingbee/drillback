# Draft issue for SiYuan - not filed

**Status: not filed, on the human's decision of 2026-09-04.** Item 1 is too thin for an issue, and item 2 is withdrawn below.

Where it would go: <https://github.com/siyuan-note/siyuan/issues>. Two small things,
neither of them a data-loss finding; a human should decide whether either is worth an
issue at all.

---

## 1 (documentation) - what to back up, and how much of it is themes

**Title:** `Docs: a line on what to back up - and that conf/appearance is 37 MB of the workspace`

**Environment:** SiYuan v3.8.2, `b3log/siyuan:v3.8.2`, Docker, workspace on a volume.

There is no backup section in the README, so a person keeps the workspace folder, which
is the right answer. What is worth a sentence is its shape. After creating one notebook
and one document:

```text
38M    workspace
 11K   workspace/data      (the notebooks and the document)
512K   workspace/temp
 37M   workspace/conf
```

and `conf/appearance` - the bundled themes, icon sets and fonts - is 37 MB of that. They
come from the image and SiYuan writes them again if they are missing: restoring
`workspace/data` alone gives a working instance with the notes in it.

Suggested addition:

> Back up the workspace folder. If you want a smaller backup, `workspace/data` holds
> your notebooks, documents and assets; `workspace/conf` is your settings plus the
> bundled themes and icons, which SiYuan writes again from the image if they are missing.

## 2 (documentation) - the Docker command in the README does not start v3.8.2

**Withdrawn on 2026-09-04.** The README on `master` now writes `serve` before
`--workspace` in both `docker run` examples and in the compose `command:`, and
[#17866](https://github.com/siyuan-note/siyuan/issues/17866), closed 2026-06-16, is the
project's own migration note for the change. The drill read a stale copy of the page.
Nothing to file. The draft is kept below as it was written, for the record.

**Title:** `README: the Docker command fails with "unknown flag: --accessAuthCode" on v3.8.2`

The README's Docker section gives:

```sh
kernel --workspace=/siyuan/workspace/ --accessAuthCode=xxx
```

On v3.8.2 that exits with:

```text
Error: unknown flag: --accessAuthCode
```

because the server moved behind a `serve` subcommand. This works:

```sh
kernel serve --workspace=/siyuan/workspace/ --accessAuthCode=xxx
```

I ran into it while restore-testing, which is the worst moment to find out that the
documented start command has changed. Happy to send a README PR if that is welcome.

---

Not raised, and deliberately: SiYuan's data repo (snapshots and cloud sync) was not
tested here at all, so nothing above is a comment on it.
