# Draft issue for Gotify - filed 2026-09-04

**Status: filed on 2026-09-04, with the human's sign-off (stop point 2), and closed
by its author the same day, with the human's sign-off:**
<https://github.com/gotify/website/issues/106>. The installation page has said "include
this directory in your backups" since July; the drill's search missed it (see
[docs.md](docs.md)). The draft is kept as filed, for the record.

Where it would go: <https://github.com/gotify/server/issues>, or the documentation
repository if the site is generated separately.

---

**Title:** `Docs: a note on what to back up - gotify.db alone leaves uploaded application icons behind`

**Environment**

- Gotify 3.1.0, `gotify/server:3.1.0`, SQLite, `/app/data` on a volume.

**What I did**

Created an application through the API, uploaded a PNG as its icon, sent one message.
Then restored two ways into a fresh instance: the whole `/app/data` directory, and
`gotify.db` on its own.

**What I observed**

The whole directory restores everything. `gotify.db` alone restores the accounts, the
applications, their tokens and the message history - and the application's icon is gone,
because it is a file:

```text
/app/data/gotify.db
/app/data/images/mlWa3Pfzp_bEro2JciusCFAJM.png
```

with `image/mlWa3Pfzp_bEro2JciusCFAJM.png` stored in the `applications` row. The
interface then shows a broken image, with nothing in the logs.

**Why this might be worth a line of documentation**

There is no backup page, and the two things the documentation does name pull in
different directions: the configuration reference names `data/gotify.db` as the database,
and the installation page mounts `/app/data`. Someone who reads the first and thinks
"the database is the data" has a backup that restores almost everything.

**Suggested change**

One line on the installation page, or a short backup section:

> Back up the whole `/app/data` directory. It holds `gotify.db`, the uploaded
> application icons in `images/`, and any plugins. If you use MySQL or PostgreSQL, back
> up that database plus `/app/data`.

Happy to send a docs PR. Gotify was the least eventful application in a set I have been
restore-testing, which is a compliment.
