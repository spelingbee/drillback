# File Browser - no issue drafted, and why

**Status: nothing to file, deliberately.**

There is a finding here, and it would normally be worth a documentation issue: the word
"backup" does not appear anywhere in File Browser's documentation, and a person who
backs up their files without `/database` loses every account, permission and share link
while getting a File Browser that starts and serves their files perfectly.

No issue is drafted, because the application prints this on every start:

```text
NOTICE: File Browser is being wound down.
NOTICE: The project is archived on 2026-09-01, after which there will be no
NOTICE: further releases and no security fixes.
```

The drill ran on 2026-08-30. Asking a maintainer to write documentation for software
they are archiving in two days is not a reasonable thing to do, and an issue filed into
a repository about to be closed is noise for whoever has to sweep up.

The finding is written for the reader instead, in [result.md](result.md). The short
version, for anyone moving off File Browser in the next few weeks - which, given the
notice, is most of the people still running it:

> Back up all three volumes. `/srv` is your files, `/database` is your users, their
> permissions and every share link, and `/config` is the daemon settings. If you restore
> `/srv` alone, File Browser will start, your files will be there, and it will create a
> new administrator whose password is printed once to the log.

Two days before a 36,000-star project is archived is exactly when people move their data
somewhere else, and moving data is a restore.
