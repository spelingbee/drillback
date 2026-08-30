# Gotify - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/gotify/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented `docker run` as compose. Deviations: no
published port, and `GOTIFY_DEFAULTUSER_NAME` / `GOTIFY_DEFAULTUSER_PASS` are set - both
documented configuration keys - so the first account has a known password instead of
`admin`/`admin`.

## 2. Seed through Gotify's own API

An application, an uploaded icon for it, and one message sent with the application's own
token.

```text
create application: 200
upload icon: 200
send message: 200
```

and what that put on disk:

```text
-rw-r--r-- 1 root root 69632 gotify.db
drwxr-x--- 2 root root  4096 images
drwxr-xr-x 2 root root  4096 plugins
--- images:
-rw-r--r-- 1 root root    70 mlWa3Pfzp_bEro2JciusCFAJM.png
```

## 3. Two backups from one copy

Reading A is `/app/data`. Reading B is `gotify.db` copied out of it on its own.

## 4. Into restic at `/app/data`, then restore each

```sh
restored check --recipe recipes/gotify --source restic --from <repo-all>
restored check --recipe recipes/gotify --source restic --from <repo-db-only>
```

Same recipe both times. Verdicts in [result.md](result.md).
