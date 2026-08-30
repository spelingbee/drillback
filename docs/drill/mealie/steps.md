# Mealie - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/mealie/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented deployment with `/app/data` on a
volume. Deviation: no published port. The account is the one Mealie creates for itself,
with the credentials its documentation gives - `changeme@example.com` / `MyPassword`.

## 2. Seed through Mealie's own API

```text
-- ready after 3 attempts
token of 185 characters
create recipe: 201
```

## 3. Back up, exactly as the page recommends

> You can easily perform entire site backups by stopping the container, and backing up
> this folder with your chosen tool. This is the best way to backup your data.

```sh
docker compose -p drill-mealie stop
docker run --rm -v drill-mealie_mealie-data:/src:ro -v <backup>/data:/dst \
  alpine:3.20 sh -c 'cp -a /src/. /dst/'
```

The directory holds `mealie.db`, `.secret`, `.session_secret`, `mealie.log`, and the
`backups/`, `groups/`, `recipes/`, `templates/` and `users/` directories.

## 4. Into restic at /app/data, then restore

```sh
restored check --recipe recipes/mealie --source restic --from <repo>
```

```text
  PASS  5/5 checks
```

Verdict in [result.md](result.md).
