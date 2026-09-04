# Draft issue for Beszel - filed 2026-09-04

**Status: filed on 2026-09-04, with the human's sign-off (stop point 2):** <https://github.com/henrygd/beszel-docs/issues/76>, on the documentation repository.
**Corrected the same day:** the last sentence of "Why I think this is worth a
documentation change" is wrong - the built-in backup feature *is* mentioned, on *What
is Beszel* and *User accounts*. What is missing is a page on what to back up by hand.
A correction comment is on the issue; the draft is kept as filed.

Where it would go: <https://github.com/henrygd/beszel/issues>.

---

**Title:** `Docs: no backup guidance, and copying data.db alone restores an empty hub that looks fine`

**Environment**

- Beszel hub 0.18.8, `henrygd/beszel:0.18.8`, the compose file from the hub installation
  page, `/beszel_data` on a volume.

**What I did**

Created the first account with the documented `USER_EMAIL` / `USER_PASSWORD`, added one
monitored system, then restored two ways into a fresh hub: the whole `/beszel_data`
directory, and `data.db` on its own.

**What I observed**

The directory restores everything. `data.db` alone restores nothing, and the failure is
very quiet - here is the state right after seeding:

```text
-rw-r--r--  4096    data.db
-rw-r--r--  32768   data.db-shm
-rw-r--r--  716912  data.db-wal
```

PocketBase runs SQLite in WAL mode, so on a young hub the file named `data.db` holds
almost nothing. Restoring it alone gives a valid, empty database, and because
`USER_EMAIL` / `USER_PASSWORD` are set the hub then creates a fresh account with the same
address and password. So the person restoring signs in successfully with their own
credentials and simply has no systems. Nothing anywhere says the restore did not work.

**Why I think this is worth a documentation change**

There is no backup page in the guide, and the hub installation page describes
`./beszel_data` as "persistent storage" without saying that it is the thing to keep. The
application also has a backup feature (PocketBase's, in Settings) that the documentation
does not mention at all.

**Suggested change**

A short *Backups* page, or a note on the hub installation page:

> Back up the whole `beszel_data` directory. It holds `data.db` and `auxiliary.db`
> together with their `-wal` files, and most recent data is in the `-wal` until SQLite
> checkpoints it - copying `data.db` alone gives you a valid but empty database. Stop
> the hub before copying, or use the backup feature in Settings.

I would be glad to send a docs PR for that, and to add a line pointing at the built-in
backup feature if that is the route you would rather people took.
