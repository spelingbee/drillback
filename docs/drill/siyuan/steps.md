# SiYuan - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/siyuan/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented Docker deployment with the workspace on
a volume. Two things had to change from the README's command, both recorded here:

| Change | Why |
|---|---|
| no published port | the drill drives SiYuan from a curl container on the project's network |
| `serve` subcommand | v3.8.2 moved the server behind it; the README's form fails with `Error: unknown flag: --accessAuthCode` |

## 2. Seed through SiYuan's API

The API token is generated on first boot and lives in `conf/conf.json`; a person copies
it out of Settings - About.

```text
api token: 16 characters
notebook: 20260830133815-2zx8udm
create document: 200
/siyuan/workspace/data/20260830133815-2zx8udm/20260830133815-zt7465x.sy
```

## 3. Back up

```text
38M    workspace
 11K   workspace/data
512K   workspace/temp
 37M   workspace/conf
```

Reading A is the workspace. Reading B is `data/` copied out of it on its own.

## 4. Into restic at /siyuan/workspace, then restore each

```sh
restored check --recipe recipes/siyuan --source restic --from <repo-workspace>
restored check --recipe recipes/siyuan --source restic --from <repo-data-only>
```

```text
  PASS  3/3 checks   <- the workspace
  PASS  3/3 checks   <- data only
```

Verdicts in [result.md](result.md).
