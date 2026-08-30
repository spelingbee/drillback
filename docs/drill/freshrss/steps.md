# FreshRSS - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/freshrss/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented Docker deployment with `data` and
`extensions` on volumes. Deviations, all recorded here:

| Deviation | Why |
|---|---|
| no published port | the drill drives it from a curl container on the project's network |
| an nginx `feed` service | one RSS feed to subscribe to that is not somebody else's site |
| `FRESHRSS_INSTALL` / `FRESHRSS_USER` | documented image variables; they install the instance and create the account so the drill knows its password |
| `CRON_MIN: ""` | no feed refresh while the drill is measuring |

## 2. Seed through FreshRSS's own CLI

`import-for-user.php` with a two-line OPML, then `actualize-user.php`.

```text
-- ready after 2 attempts
FreshRSS actualized 1 feeds for drilladmin (1 new articles)
-rw-rw-r-- 1 root www-data 520192 db.sqlite
```

## 3. Back up, as the page says

```text
1.3M   data
8.0K   extensions
581K   data without cache/
```

## 4. Into restic at the documented paths, then restore each

```sh
restored check --recipe recipes/freshrss --source restic --from <repo-all>
restored check --recipe recipes/freshrss --source restic --from <repo-no-cache>
```

```text
  PASS  5/5 checks   <- data and extensions
  PASS  5/5 checks   <- the same without cache/
```

Verdicts in [result.md](result.md).
