# File Browser - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/filebrowser/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented `docker run` as compose: the bare
Alpine image with the three documented volumes. One deviation: no published port.

The first-boot log is worth keeping, because it is the only place the password is ever
printed:

```text
filebrowser-1 | User 'admin' initialized with randomly generated password: TJHWNwg0mJQPWOX6
filebrowser-1 | NOTICE: File Browser is being wound down.
filebrowser-1 | NOTICE: The project is archived on 2026-09-01, after which there will be no
```

## 2. Seed through File Browser's own API

Sign in as the bootstrapped admin with the password from that log, upload a file, and
share it. The share is the piece that lives in the database rather than on disk.

```sh
TOKEN=$(curl -X POST http://filebrowser:80/api/login \
  -d '{"username":"admin","password":"<from the log>","recaptcha":""}')
curl -X POST "http://filebrowser:80/api/resources/drill-canary.txt?override=true" \
  -H "X-Auth: $TOKEN" --data-binary 'drill canary file: this line proves the restore'
curl -X POST "http://filebrowser:80/api/share/drill-canary.txt" \
  -H "X-Auth: $TOKEN" -d '{"password":"","expires":"","unit":"hours"}'
```

```text
login: token of 603 characters
upload: 200
share: 200
```

## 3. Back up the three documented volumes

```sh
docker run --rm -v drill-filebrowser_filebrowser_data:/src:ro     -v <backup>/data:/dst     alpine:3.20 sh -c 'cp -a /src/. /dst/'
docker run --rm -v drill-filebrowser_filebrowser_database:/src:ro -v <backup>/database:/dst alpine:3.20 sh -c 'cp -a /src/. /dst/'
docker run --rm -v drill-filebrowser_filebrowser_config:/src:ro   -v <backup>/config:/dst   alpine:3.20 sh -c 'cp -a /src/. /dst/'
```

## 4. Into restic at the three documented paths

```sh
restic backup /srv /database /config
```

Reading B is the same command with an empty directory bound at `/database`, so the
snapshot has the path and nothing in it - which is what a backup of a named volume that
was never mounted looks like.

## 5. Tear down, then restore each

```sh
restored check --recipe recipes/filebrowser --source restic --from <repo-all>
restored check --recipe recipes/filebrowser --source restic --from <repo-no-db>
```

Same recipe both times. Verdicts in [result.md](result.md).
