# The official-docs restore drill

For each popular self-hosted application: follow its **own backup documentation
literally**, take the backup that documentation describes, restore it with `restored`,
and record whether the application comes back with its data.

The finding is never "application X loses data". It is "if you follow the documentation
as written, here is what you get back". A gap in the documentation is the expected
result, and each one is written up as a respectful draft issue that a human reviews and
files - or does not.

Rules this drill runs under, in full, are in the session brief; the ones that matter to a
reader:

- The backup is **exactly** what the documentation says. Improving it defeats the study.
- Every FAIL and PARTIAL is reproduced twice from an empty scratch directory. No FAIL
  without logs.
- Ambiguity in the documentation is itself a finding. Where a page can be read two ways,
  the most obvious reading is the verdict and the other one is tested too when it is
  cheap.
- **Nothing is filed anywhere.** `upstream-issue.md` in each folder is a draft.

## Results

| App | Stars | Docs | Version | Result | One-line cause |
|---|---|---|---|---|---|
| [n8n](n8n/) | 202,807 | [CLI page](https://docs.n8n.io/deploy/host-n8n/configure-n8n/use-the-command-line) | 2.36.8 | **PARTIAL** | The only thing the docs call a backup - `export:workflow/credentials --backup` - has no users in it and no encryption key, so the restored instance asks you to create an owner and cannot decrypt a credential. Backing up `.n8n` instead: PASS. |
| [memos](memos/) | 62,640 | [Docker Compose](https://usememos.com/docs/deploy/docker-compose) | 0.30.0 | **PASS** / **FAIL** | Copying the data directory restores everything. Copying the file the docs call "the database" restores an empty Memos: memos_prod.db was 4 KiB and its -wal was 160 KiB. |

Verdicts:

- **PASS** - the documented backup restores an application with its data in it.
- **PARTIAL** - it boots, and something a person would expect to have is missing.
- **FAIL** - it does not come back.
- **SKIPPED** - not testable inside the drill's budget or isolation rules. The reason is
  recorded; a skip is data too.

## What is in each folder

| File | What it is |
|---|---|
| `docs.md` | The official documentation: URL, version, and the steps quoted as written. |
| `compose.yaml` | The application deployed the way its own documentation says to. |
| `run.sh` | Every command the drill ran, in order. This is the file that was executed. |
| `steps.md` | The same commands with their real output, and every deviation named. |
| `recipe/` | The recipe used against the documented backup, when it differs from the registry one. |
| `result.md` | The verdict, the report, the root cause with evidence, and what the docs would need to say. |
| `result-*.txt`, `result-*.json` | The reports, written by the tool. Never by hand. |
| `upstream-issue.md` | A draft issue. Not filed. |

## Running one

From the repository root, with docker and restic on PATH:

```sh
bash docs/drill/n8n/run.sh
```

Shared helpers are in [`lib.sh`](lib.sh). Every application gets its own compose project
named `drill-<app>`, and it is torn down with its volumes at the end of the run.
