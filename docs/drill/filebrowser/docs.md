# File Browser - the official backup documentation

- **Application:** [filebrowser/filebrowser](https://github.com/filebrowser/filebrowser),
  35,968 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v2.63.23. Image `filebrowser/filebrowser:v2.63.23`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No, and there is no mention of backups anywhere. The documentation now lives in the
repository's `docs/` directory - `filebrowser.org/installation` redirects to the GitHub
repository - and the word does not appear in any of it:

```text
docs/README.md            no match
docs/installation.md      no match
docs/deployment.md        no match
docs/troubleshooting.md   no match
docs/customization.md     no match
docs/authentication.md    no match
```

(`grep -in "backup\|back up"` over each file, 2026-08-30.)

## What the documentation does say

From `docs/installation.md`, the documented Docker command, verbatim:

```sh
docker run \
    -v filebrowser_data:/srv \
    -v filebrowser_database:/database \
    -v filebrowser_config:/config \
    -p 8080:80 \
    filebrowser/filebrowser
```

> Where `filebrowser_data`, `filebrowser_database` and `filebrowser_config` are Docker
> volumes, where the data, database and configuration will be stored, respectively.

and, for the s6 image, what is in each:

> - `/path/to/srv` contains the files root directory for File Browser
> - `/path/to/config` contains a `settings.json` file
> - `/path/to/database` contains a `filebrowser.db` file

Three volumes, each named. That is the whole of the guidance a person has, and it is the
primary reading: keep all three.

The other sentence that matters is a warning, not backup advice, and it is the reason
losing the database is worse than it sounds:

> **Warning:** The automatically generated password for the user `admin` is only
> displayed once. If you fail to remember it, you will need to manually delete the
> database and start File Browser again.

## One thing that should be recorded plainly

The application prints this on every start, and it printed it during this drill:

```text
NOTICE: File Browser is being wound down.
NOTICE: The project is archived on 2026-09-01, after which there will be no
NOTICE: further releases and no security fixes. Known unfixed issues are at
NOTICE: https://github.com/filebrowser/filebrowser/security/advisories
```

The drill ran on 2026-08-30. Two days before a 36,000-star project is archived is
exactly when people move their data somewhere else, and moving data is a restore. No
upstream issue is drafted for this application: there is no one left to receive it, and
filing documentation requests against a repository that is being archived would be
inconsiderate. The finding is written up for the reader instead.

## The two readings

- **A: all three volumes.** What the installation page describes.
- **B: the files, without the database.** For a file manager, "I have a backup of my
  files" is an easy thing to believe. `/srv` is the files; `/database` is everything
  else.
