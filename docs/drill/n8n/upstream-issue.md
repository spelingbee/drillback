# Draft issue for n8n - filed 2026-09-04

**Status: filed on 2026-09-04, with the human's sign-off (stop point 2).** Issue 2 is <https://github.com/n8n-io/n8n/issues/37814>, in the shape of the repository's bug form; issue 1 is <https://github.com/n8n-io/n8n-docs/issues/5325>, on the documentation repository, and links the bug.

Where it would go: <https://github.com/n8n-io/n8n-docs/issues> (the documentation repo),
or n8n's docs feedback route if that is preferred. Two separate issues are proposed,
because one is a documentation gap and the other is a behaviour bug.

---

## Issue 1 (documentation) - Docs: self-hosted backup guidance, and `--backup` does not include the encryption key

**Title:** `Docs: what "backup" covers for a self-hosted instance (export --backup vs the .n8n directory)`

**Environment**

- n8n 2.36.8, `n8nio/n8n:2.36.8`, Docker, default SQLite database.
- Deployed with the command from
  [Install with Docker](https://docs.n8n.io/deploy/host-n8n/install-options/install-with-docker).

**What I was looking for**

How to back up a self-hosted n8n. Searching the docs sitemap for "backup" finds no page
about it; the only place the word appears for self-hosting is the `--backup` flag on
`export:workflow` and `export:credentials` in
[Use the command line](https://docs.n8n.io/deploy/host-n8n/configure-n8n/use-the-command-line).
So that is what I used.

**What I did**

1. Started n8n with the documented Docker command.
2. Created an owner account, one workflow, and one credential (`httpHeaderAuth`) through
   the UI's API.
3. Ran the documented backup, exactly as the page's examples write it:

   ```sh
   n8n export:workflow    --backup --output=backups/latest/
   n8n export:credentials --backup --output=backups/latest/
   ```

4. Destroyed the instance and its volume.
5. Started a fresh instance and ran the documented import:

   ```sh
   n8n import:workflow    --separate --input=backups/latest/
   n8n import:credentials --separate --input=backups/latest/
   ```

**What I observed**

- The workflow came back.
- The instance had no users: it showed the first-run owner-setup screen, so anybody who
  reaches it first becomes the owner.
- The credential could not be decrypted, because the exported `data` field is ciphertext
  under the key in `.n8n/config`, and `--backup` does not write that file.

**Why I think this is worth a documentation change**

The `--decrypted` flag's description already tells the reader that plaintext is what you
need when moving "to another installation that has a different secret key". That is the
same fact a person needs when *restoring*, but it is written as migration advice, and
`--decrypted` is not in the `--backup` examples. Someone who follows the `--backup`
examples has a file set that looks like a backup, imports without complaint (see issue 2
for the exception), and is missing their login and their secrets.

**Suggested change**

A short *Back up and restore* page under *Host n8n*, or a note on the CLI page, saying:

- what the `--backup` exports do and do not contain (no users, executions, variables, or
  encryption key);
- that a complete backup of a self-hosted instance is the `.n8n` directory - including
  the `config` file that holds the encryption key - plus the external database if one is
  configured;
- that restoring credentials into a different instance needs either the same `config`
  file or `--decrypted` at export time, with the obvious warning about what a plaintext
  export contains.

I am happy to open a PR against n8n-docs with a draft of that page if it would be
useful - please say which shape you would prefer.

---

## Issue 2 (behaviour) - `import:credentials --separate` fails on the directory `--backup` writes

**Title:** `import:credentials --separate fails with SQLITE_CONSTRAINT when the directory also contains exported workflows`

**Environment**

- n8n 2.36.8, `n8nio/n8n:2.36.8`, Docker, SQLite. Reproduced again on 2.37.9
  (`n8nio/n8n:2.37.9`, released 2026-09-03) on 2026-09-04, with the same output.

**Steps to reproduce**

The docs' own examples write both exports into one directory:

```sh
n8n export:workflow    --backup --output=backups/latest/
n8n export:credentials --backup --output=backups/latest/
```

Then, on a fresh instance:

```sh
n8n import:workflow    --separate --input=backups/latest/   # ok
n8n import:credentials --separate --input=backups/latest/   # fails
```

**Observed**

```text
$ n8n import:workflow --separate --input=/restore
Skipping invalid workflow file: /restore/<credential-id>.json
Importing 1 workflows...
Successfully imported 1 workflow.

$ n8n import:credentials --separate --input=/restore
An error occurred while importing credentials. See log messages for details.
SQLITE_CONSTRAINT: NOT NULL constraint failed: credentials_entity.data
```

Control: the same `import:credentials` command against a directory holding only the
credential file succeeds ("Successfully imported 1 credential."). So the failure is
caused by the exported *workflow* file sitting in the directory, which
`import:credentials --separate` reads and tries to insert as a credential.

**Expected**

Either `import:credentials --separate` skips files that are not credentials, the way
`import:workflow --separate` already skips files that are not workflows, or the
documentation's examples write the two exports to different directories.

**Impact**

The import aborts entirely, so the credentials that *would* have imported are not
imported either. Following the documented commands as written, a person restoring gets
their workflows and none of their credentials, with an error that reads like database
corruption rather than "wrong file in the directory".

I would be glad to send a PR for whichever fix you prefer - the skip in
`import:credentials`, or a docs change to use two directories.
