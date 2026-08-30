# Draft issue for ConvertX - NOT FILED

**Status: draft. Nothing has been filed (CLAUDE.md stop point 2).**

Where it would go: <https://github.com/C4illin/ConvertX/issues>.

---

**Title:** `Docs: a line on what to back up - mydb.sqlite alone restores an instance you cannot log in to`

**Environment**

- ConvertX v0.18.0, `ghcr.io/c4illin/convertx:v0.18.0`, the compose file from the
  README, `/app/data` on a volume, `JWT_SECRET` set.

**What I did**

Registered the first account, loaded the page, uploaded a file. Then restored two ways
into a fresh instance: the whole `/app/data` directory, and `mydb.sqlite` on its own.

**What I observed**

The directory restores everything - the account, the job, the uploaded file, and a
sign-in that works.

`mydb.sqlite` alone restores an instance where `POST /login` answers **403**. Two
separate reasons:

1. The rows were in the write-ahead log. After registering one account and uploading one
   file:

   ```text
   -rw-r--r--  20480  mydb.sqlite
   -rw-r--r--  32768  mydb.sqlite-shm
   -rw-r--r--  16512  mydb.sqlite-wal
   ```

   so a copy of `mydb.sqlite` is a copy of the database as it was at the last
   checkpoint.

2. The uploaded and converted files are not in the database at all - they are in
   `uploads/<user>/<job>/` and `output/<user>/<job>/`, and the `jobs` rows point at them
   by user and job id.

Because only the first account can register, the restored instance is then sitting on
the setup page waiting for whoever reaches it first - which the README already warns
about in a different context ("Don't leave it unconfigured and open, as anyone can
register the first account").

**Suggested change**

One line under Deployment:

> Back up the whole `data` directory. It holds `mydb.sqlite` - together with its `-wal`
> file, where recent rows live until SQLite checkpoints them - and the `uploads/` and
> `output/` directories holding the files your conversion jobs refer to. Copying
> `mydb.sqlite` alone gives you an instance you cannot log in to.

I would be glad to send a README PR with that. Thank you for ConvertX - the deployment
is one compose file and one volume, which is exactly why the one line is worth having.
