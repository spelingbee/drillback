# n8n - the official backup documentation

- **Application:** [n8n-io/n8n](https://github.com/n8n-io/n8n), 202,807 stars at the time
  `docs/recipes-wanted.txt` was gathered (2026-08-30).
- **Version tested:** 2.36.8, the version the documentation site names as *Current
  stable* on the day of the drill. Image `n8nio/n8n:2.36.8`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No. This is the first finding, and it shapes everything below.

`https://docs.n8n.io/sitemap.md` was fetched and searched for pages whose title or path
mentions backup, restore, export, import, cli or database. The complete result:

```text
/deploy/use-n8n-cloud/download-workflows.md
/deploy/host-n8n/configure-n8n/choose-n8ns-database.md
/deploy/host-n8n/configure-n8n/basic-configuration/use-environment-variables/database.md
/deploy/host-n8n/configure-n8n/use-the-command-line.md
/deploy/host-n8n/understand-the-architecture/understand-the-database.md
/build/manage-workflows/export-and-import.md
/build/manage-workflows/n8n-packages/export-a-package.md
/build/manage-workflows/n8n-packages/import-a-package.md
```

There is no *Back up your n8n* page for a self-hosted instance. A person looking for one
finds two things instead, and they are the two readings this drill tested.

## Reading A - the only thing the documentation calls a backup

<https://docs.n8n.io/deploy/host-n8n/configure-n8n/use-the-command-line.md>

The page documents a `--backup` flag on two commands, and gives these examples verbatim:

```text
n8n export:workflow --backup --output=backups/latest/
n8n export:credentials --backup --output=backups/latest/
```

with the flag described as:

> `--backup`: "Sets --all --pretty --separate for backups"

and the matching import examples, again into and out of one directory:

```text
n8n import:workflow --separate --input=backups/latest/
n8n import:credentials --separate --input=backups/latest/
```

The page also documents `--decrypted` on the credentials export:

> "Export all the credentials in plain text format. You can use this to migrate from one
> installation to another that has a different secret key in the configuration file."

That sentence is the only warning anywhere near this pair of commands that a credential
exported *without* `--decrypted` is tied to the instance's key. It is written as advice
for migrating, not as a caveat about backups.

## Reading B - what the installation page says to keep

<https://docs.n8n.io/deploy/host-n8n/install-options/install-with-docker>

The documented Docker command, quoted in full:

```shell
docker volume create n8n_data

docker run -it --rm \
 --name n8n \
 -p 5678:5678 \
 -e GENERIC_TIMEZONE="<YOUR_TIMEZONE>" \
 -e TZ="<YOUR_TIMEZONE>" \
 -e N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true \
 -e N8N_RUNNERS_ENABLED=true \
 -v n8n_data:/home/node/.n8n \
 n8nio/n8n
```

and, about that volume:

> "When using PostgreSQL, n8n doesn't need to use the `.n8n` directory for the SQLite
> database", but "the directory still contains other important data like encryption
> keys, instance logs, and source control feature assets."

> it's best to "continue mapping a persistent volume for the directory to avoid
> potential issues"

The page says to *keep* the directory. It does not say to copy it anywhere, and it never
uses the word backup.

## The ambiguity, stated plainly

"How do I back up n8n?" has two documented answers that produce very different files:
the export pair, which is what the documentation calls a backup, and the `.n8n`
directory, which is what the documentation says holds the data. Reading A was taken as
the primary reading because it is the one the word "backup" points at. Reading B was
tested too, because it is cheap and because the difference between them is the finding.
