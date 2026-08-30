# n8n - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the documented export | `n8n export:workflow --backup` + `n8n export:credentials --backup` into one directory | **PARTIAL** (`RESTORE UNUSABLE`, exit 1, 2 of 4 checks) | [result-export.txt](result-export.txt) |
| B - the data directory | `/home/node/.n8n`, which the installation page says to keep | **PASS** (exit 0, 6 of 6 checks) | [result-dotn8n.txt](result-dotn8n.txt) |

The headline is not "n8n loses data". It is: **the two documented answers to "how do I
back up n8n" give back different things, and the one the documentation calls a backup
comes back without the login and without a usable credential.**

## Reading A, in full

```text
restored 0.1.0-dev · recipe n8n-documented-export · run ycuaux3v

  source     restic  C:/Users/kadyr/AppData/Local/Temp/restored-drill/n8n/restic-export
  snapshot   af70d8e8  2026-08-30 11:19:59  host=drill  tags=[drill]
  inputs     export  /home/node/backups/latest    1.9 KiB  2 files

  restore    ok          1.7s   1 input
  compose    ok         15.7s   3 services
  ready      ok          8.5s   n8n reports itself ready

  CHECKS
  ✔  healthz                 n8n answers its own health endpoint         0.54s
  ✔  workflow-restored       The workflow that was backed up is in the    3.3s
                             restored instance
  ✘  owner-account-restored  The instance still has its owner account,   0.68s
                             so somebody can log in
                               expect  body_matches: "showSetupOnFirstLoad"\s*:\s*false
                               got     {"data":{"settingsMode":"public","defaultLocale":"en","userManagement":{"authenticationMethod":"email","showSetupOnFirstLoad":true,"smtpSetup":false,"passwordMinLength":8},"sso":{"saml":{"loginEnabled":false},"ldap":{"loginEnabled":false,"loginLabel":""},"oidc":{"loginEnabled":false,"loginUrl":"http...
  ✘  credential-decryptable  A restored credential can still be           3.2s
                             decrypted
                               expect  exit_code: 0
                               got     exit_code: 1

  RESTORE UNUSABLE  2/4 checks  ·  total 35.5s  ·  teardown ok

  Service logs from the failure window are in the JSON report (--report).
  Re-run with --keep to keep the stack up and poke at it yourself.
```

What that means for a person: n8n starts, the workflow is there, and the browser shows
the first-run *set up your owner account* screen. Whoever reaches the restored instance
first becomes its owner. The credential is either absent or unreadable.

## Root causes

### 1. The documented backup contains workflows and credentials, and nothing else

`--backup` "sets `--all --pretty --separate`". `--all` means all workflows, or all
credentials - not all state. What is therefore never in the file set: the `user` table
(so no login), executions, variables, tags, projects, settings, and - decisively - the
`.n8n/config` file that holds the instance's encryption key.

Evidence, the whole of the credential file the documented export wrote:

```json
{
  "updatedAt": "2026-08-30T10:47:14.613Z",
  "createdAt": "2026-08-30T10:47:14.615Z",
  "id": "nW7gKEoPV7OBsdDe",
  "name": "drill-canary-credential",
  "data": "U2FsdGVkX190/bYBza6CZREYK+sF04z9mFZ9viF6oNJU1pvaLLmxESpHagHAiwsiDZLsDxiHNvHEJ6eHc9EodA==",
  "type": "httpHeaderAuth",
  "isManaged": false,
  "isGlobal": false
}
```

`data` is ciphertext under a key that lives in `.n8n/config`. Restoring this file into a
new instance restores a string nobody can read. The documentation does say, on the
`--decrypted` flag, that plain text is what you need "to migrate ... to another
[installation] that has a different secret key" - but that sentence is filed under
migration, and the flag is not in the `--backup` examples.

### 2. `import:credentials` cannot read the directory `export:credentials --backup` wrote

This one is a hard failure, and it is entirely inside the documentation's own examples.
The export examples write both kinds of file into `backups/latest/`. The import examples
read that same directory. `import:workflow` copes - it skips what it does not
understand. `import:credentials` does not.

Reproduced three times: inside the restored stack, and twice standalone with a control.

```text
$ docker run --rm -v <export-dir>:/restore:ro n8nio/n8n:2.36.8 \
    import:workflow --separate --input=/restore
Skipping invalid workflow file: /restore/nW7gKEoPV7OBsdDe.json
Importing 1 workflows...
Successfully imported 1 workflow.

$ docker run --rm -v <export-dir>:/restore:ro n8nio/n8n:2.36.8 \
    import:credentials --separate --input=/restore
An error occurred while importing credentials. See log messages for details.
SQLITE_CONSTRAINT: NOT NULL constraint failed: credentials_entity.data

# control: the same command against a directory holding ONLY the credential file
$ docker run --rm -v <creds-only>:/restore:ro n8nio/n8n:2.36.8 \
    import:credentials --separate --input=/restore
Successfully imported 1 credential.
```

The cause is not the credential file. It is the workflow file sitting next to it:
`import:credentials --separate` reads every `*.json` in the directory and inserts it as a
credential, and a workflow document has no value for the `data` column. The whole import
aborts, so even the credential that *would* have imported does not.

The two documented commands are therefore not each other's inverse for the directory
layout the documentation itself specifies.

### 3. Reading B round-trips, and the file that makes the difference is `config`

The `.n8n` directory restores to a working instance: owner present, workflow present,
credential decryptable, `PRAGMA integrity_check` clean. The recipe in
[`recipes/n8n`](../../../recipes/n8n) asserts all four, and its `credential-decryptable`
check exists precisely to fail if somebody backs up `database.sqlite` alone - a 56-byte
file called `config`, sitting beside a 1.5 MB database, is the difference between a
credential and a base64 string.

### 4. An operational trap worth its own line: n8n's startup server answers 200

Not a backup finding, but it cost this drill an hour and it will cost anybody who
automates n8n the same hour. While n8n starts, a placeholder server answers - and it
answers `POST /rest/owner/setup` with **200 and a body that creates nothing**.
`/rest/settings` answers 200 from that server too. Only `/healthz/readiness` tells the
truth. Measured on 2.36.8, polling once a second from container start:

```text
t=4  healthz=200 readiness=503 settings=200 ownersetup=200   <- placeholder server
t=7  healthz=200 readiness=503 settings=404 ownersetup=404   <- handover
t=9  healthz=200 readiness=503 settings=404 ownersetup=400
t=10 healthz=200 readiness=200 settings=200 ownersetup=400   <- real server
```

An account created in the `t=4..6` window is silently discarded. Both recipes here wait
on `/healthz/readiness`, and the seed step checks that the response body contains the
address it asked for rather than trusting the status code.

## Ambiguity, recorded

"Back up n8n" has two documented readings; both were tested; they disagree. Reading A is
what the word "backup" points at in the documentation and is the primary reading here.
Reading B is what the installation page's advice implies, and it is the one that works.
A person who read only the CLI page would have reading A.

## What the documentation would need to say instead

A page under *Host n8n* called *Back up and restore*, saying:

1. Back up the `.n8n` directory - the database **and** `config`, which holds the
   encryption key - or, with PostgreSQL, the database plus `config`.
2. If you use `export:workflow --backup` and `export:credentials --backup`, say what
   they do not contain: users, executions, variables, and the key that makes the
   credentials readable.
3. Write the two exports to **different directories**, because
   `import:credentials --separate` fails on a directory that also holds workflow files.

## Not tested

- **Executions and variables.** Neither was seeded: n8n has no documented CLI that
  creates an execution, and variables are a licensed feature. Neither appears in the
  documented export, but this drill did not demonstrate their loss and does not claim it.
- **PostgreSQL.** Only the default SQLite deployment was tested. With PostgreSQL the
  `.n8n` directory still holds the encryption key, so finding 3 gets sharper, not softer -
  but that was not measured here.
- **`--decrypted` as a backup strategy.** Not tested. It would put every secret the
  instance holds in plain text in the backup, which is a trade a person should make
  knowingly.

## Reproduced

Both verdicts were produced twice, from an empty scratch directory each time: once while
the recipes were being written (2026-08-30 10:49 UTC) and once from `run.sh` end to end
(2026-08-30 11:19 UTC). The failing checks and the exit codes were identical.
