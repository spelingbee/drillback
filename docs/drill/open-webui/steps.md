# Open WebUI - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/open-webui/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented quick start as a compose file: the
image and a volume on `/app/backend/data`. One deviation: no published port, because
the drill drives the instance from a curl container on the project's own network.

`WEBUI_SECRET_KEY` is set to a fixed value. It signs session tokens and encrypts
nothing in the database, so it does not affect what a restore gives back; it is pinned
only so two runs of the same drill agree.

```text
-- ready after 31s-ish: http://open-webui:8080/health -> 200
```

## 2. Seed through Open WebUI's own API

One account and one saved chat. Both calls run from inside the container, because the
chat endpoint needs the bearer token that sign-up returns.

```python
user = post('/api/v1/auths/signup', {'name': 'Drill Operator',
    'email': 'drill@example.invalid', 'password': 'Drill-Password-1'})
chat = post('/api/v1/chats/new', {'chat': {
    'title': 'drill-canary-chat', 'models': ['drill-model'],
    'messages': [{'id': 'drill-msg-1', 'role': 'user', 'content': 'drill canary message'}]}},
    user['token'])
```

```text
signed up drill@example.invalid - saved chat 7f594cc9-674e-4f72-a612-28529adc5e3c drill-canary-chat
```

## 3. Back up, as documented

The backup page's scripts bring the stack down before copying, so the drill stops the
container and copies the volume with nothing writing to it:

```sh
docker compose -p drill-open-webui stop
docker run --rm -v drill-open-webui_open-webui:/src:ro -v <backup>/data:/dst \
  alpine:3.20 sh -c 'cp -a /src/. /dst/'
```

```text
-- what the documented backup weighs, and where the weight is:
1.1G    backup/data
   0    backup/data/uploads
 32K    backup/data/webui.db-shm
160K    backup/data/webui.db-wal
184K    backup/data/vector_db
632K    backup/data/webui.db
1.1G    backup/data/cache
-- symlinks in it: 46
```

## 4. Two snapshots from one copy

Reading A is that directory. Reading B is the same directory with `cache/` left out -
a deviation from the page's list of five, made because reading A cannot be restored on
this host and labelled as a deviation in [result.md](result.md).

```sh
(cd <backup>/data && tar cf - --exclude=./cache .) | (cd <backup>/data-no-cache && tar xf -)
```

```text
1012K   backup/data-no-cache
```

Both go into their own restic repository at `/opt/open-webui`, which is the host path
the documentation's own bind-mount example uses.

## 5. Restore each

```sh
restored check --recipe recipes/open-webui --source restic --from <repo-full>
restored check --recipe recipes/open-webui --source restic --from <repo-no-cache>
```

Same recipe both times; the only difference is what is in the snapshot. Verdicts in
[result.md](result.md).
