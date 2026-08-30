# Running `drillback` from a container

There is an image that bundles `drillback`, the Docker CLI, the Compose plugin and
`restic`, for machines where installing four things by hand is the hard part - which
is most NAS boxes.

```
ghcr.io/spelingbee/drillback:0.1.0
```

**Read the security section before you run it.** This image talks to your Docker daemon
through a mounted socket, and a mounted Docker socket is root on the host. That is not
a warning about this tool in particular; it is what the socket is.

---

## The invocation

Two things about it are not obvious, and both are load-bearing.

```sh
# Read the gid of your docker socket once. It is usually the `docker` group.
DOCKER_GID=$(stat -c %g /var/run/docker.sock)

# The workspace directory has to exist on the host, at the same path it will have
# inside the container. See "The same-path rule" below.
sudo mkdir -p /var/lib/drillback

docker run --rm \
  --group-add "$DOCKER_GID" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/drillback:/var/lib/drillback \
  -v /mnt/backups/restic:/mnt/backups/restic:ro \
  -e RESTIC_PASSWORD_FILE=/run/secrets/restic-pass \
  -v /etc/restic/pass:/run/secrets/restic-pass:ro \
  ghcr.io/spelingbee/drillback:0.1.0 \
    check --recipe gitea \
      --source restic --from /mnt/backups/restic \
      --workspace /var/lib/drillback \
      --input data=/srv/gitea/data \
      --input db=/srv/gitea/db.sql
```

### The same-path rule

`--workspace /var/lib/drillback` and `-v /var/lib/drillback:/var/lib/drillback` are the
same path on both sides on purpose, and the run will not work otherwise.

`drillback` restores your backup into the workspace and then asks the daemon to
bind-mount parts of it into the recipe's containers. **The daemon resolves those paths
on the host, not inside this container.** If the workspace only exists inside the
container, the daemon is asked to mount a path that, as far as it is concerned, does
not exist - and Docker's answer to that is to create an empty directory and mount that.
The application starts against nothing, every check fails, and the report says
`RESTORE UNUSABLE` about a backup that was fine.

So: same path inside and out, and it must exist on the host before the run.

The same reasoning applies to `--from`. The repository is read by `restic` *inside* this
container, so that one only needs to be mounted somewhere; it does not need to match. It
is mounted read-only above because a restore drill has no business writing to your
backup repository, and this is the cheapest place to say so.

### Which parts need which mount

| mount | why |
|---|---|
| `/var/run/docker.sock` | drillback has no daemon of its own. This is the whole mechanism, and the whole risk. |
| `/var/lib/drillback` (same path both sides) | the drillback data, which the daemon must be able to see. |
| your restic repository | read by `restic` inside the container. Read-only. |
| your restic password file | never passed as an argument; `RESTIC_PASSWORD_FILE` points at it. |

### Reading the password

Use `RESTIC_PASSWORD_FILE` and mount the file, as above. Do not use `-e
RESTIC_PASSWORD=...`: an environment variable set on the command line is visible in
your shell history, in `docker inspect`, and to anything that can read the container's
environment. `drillback` never logs restic's environment and never parses it, but it
cannot un-leak a value the daemon is already holding.

If your repository URL itself carries a password - `rest:https://user:pass@host/repo` -
`drillback` scrubs it out of the report and the debug log before you see them (ADR-059).
It cannot scrub it out of your shell history.

---

## Security: what mounting the socket actually grants

**A container with the Docker socket can become root on the host.** It can start a new
container with `--privileged`, or one that bind-mounts `/` and edits `/etc/shadow`.
There is no configuration of this image, and no setting inside `drillback`, that
prevents that; the grant is made by the `-v /var/run/docker.sock` on your command line,
not by anything in here.

That is worth stating plainly, because `drillback`'s other security properties are
strong and it would be easy to read them as covering this. They do not. The isolation
rules in SPEC.md section 9 constrain what a **recipe** may ask the daemon for - no
privileged containers, no host namespaces, no published ports, no bind mount outside
the run workspace, and the socket is never mounted *into* a recipe's container. They
say nothing about what **you** grant this image, because you granted it before
`drillback` started.

So the useful question is not "is this safe" but "is this a smaller grant than the
alternative", and it is:

- **You already trust this daemon** with the containers you run on it. `drillback` is
  one more.
- **The recipes are the untrusted part**, and they are constrained by a schema the
  daemon never sees around - an allow-list, so a compose key nobody has considered is
  rejected rather than granted (ADR-057).
- **The image is not the boundary.** Running `drillback` as a binary on the host, which
  is what `install.sh` gives you, grants exactly the same thing, because it needs the
  same daemon. The container adds no risk over the native install; it just makes the
  grant visible on the command line.

### Reducing it

- **Do not run this on a machine you would not run your backups on.** It is already
  reading them.
- **`--group-add` rather than `--user 0`.** The image runs as uid 65532 and cannot read
  the socket by default, which is deliberate: adding the socket's group is a decision
  you make once and can see in the command. On a host where the socket is owned by
  `root:root` with mode 660 - Docker Desktop, for instance - there is no difference,
  and you should know that rather than assume otherwise.
- **Mount the repository read-only.** A restore drill never writes to a backup.
- **Prefer a rootless daemon or a socket proxy** if you have one. `drillback` needs
  `containers`, `images`, `networks`, `volumes` and `exec`; a proxy that grants only
  those is a real reduction, and it is the configuration to reach for on a machine that
  runs anything else important.

---

## Scheduling it

The exit code is the whole interface, so anything that can run a container on a timer
will do.

```sh
# cron, weekly, with the report kept
0 4 * * 0 docker run --rm --group-add "$(stat -c %g /var/run/docker.sock)" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/drillback:/var/lib/drillback \
  -v /mnt/backups/restic:/mnt/backups/restic:ro \
  -v /etc/restic/pass:/run/secrets/restic-pass:ro \
  -e RESTIC_PASSWORD_FILE=/run/secrets/restic-pass \
  -v /var/log/drillback:/var/log/drillback \
  ghcr.io/spelingbee/drillback:0.1.0 \
    check --recipe gitea --source restic --from /mnt/backups/restic \
      --workspace /var/lib/drillback \
      --report /var/log/drillback/gitea-$(date +\%F).json
```

The exit code is the thing to alert on: **0** means the restore worked, **1** means it
did not, **2** means the drill could not be completed and says nothing about your
backup. Alert on 1 loudly and on 2 quietly; a 2 is usually a machine problem.

`--report` writes the JSON document. `schema_version` is `1`, and within a major
version fields are only ever added, so it is safe to script against.

---

## Building it yourself

```sh
git clone https://github.com/spelingbee/drillback
cd drillback
docker build -t drillback:dev .
docker run --rm drillback:dev version
```

The build is a two-stage `golang:1.27-alpine` to `alpine:3.22`, and the runtime layer is
`drillback`, `docker-cli`, `docker-cli-compose`, `restic` and `ca-certificates`. Nothing
else.

---

## What was verified, and what was not

Because a document that says "this works" without saying how it was checked is the kind
of thing this project does not ship.

Verified on the machine that wrote this (Docker Desktop 29.5.2, Linux containers):

```text
$ docker run --rm drillback:dev version
drillback 0.0.0-docker
  docker:    not found
  restic:    0.18.0
  recipes:   5 bundled
exit=0

$ docker run --rm --group-add 0 -v /var/run/docker.sock:/var/run/docker.sock drillback:dev version
  docker:    29.5.2 (compose v2.36.2)
  restic:    0.18.0

$ docker run --rm -v /var/run/docker.sock:/var/run/docker.sock drillback:dev version
  docker:    not found          # the non-root user cannot read the socket without --group-add
```

**A full `drillback check` from inside the image, against the host daemon.** Docker
Desktop's daemon lives in its own VM, so the same-path rule needed a path that both the
container and that VM resolve identically - which is the same thing it needs on a NAS,
arrived at differently:

```text
$ docker run --rm -v /:/host alpine:3.22 mkdir -p /host/var/lib/drillback-demo

$ docker run --rm -u 0     -v /var/run/docker.sock:/var/run/docker.sock     -v /var/lib/drillback-demo:/var/lib/drillback-demo     -v /path/to/drillback:/src:ro -v /path/to/release:/rel:ro     -e TMPDIR=/var/lib/drillback-demo -e DRILLBACK_BIN=/rel/drillback     -w /src --entrypoint sh drillback:dev -c './scripts/demo.sh'

drillback 0.0.1-snapshot - recipe gitea - run ffaczssj
  source     restic  /var/lib/drillback-demo/tmp.jFkOFD/repo
  snapshot   1961a4a9  2026-08-30 02:38:11  host=demo-host  tags=[gitea]
  restore    ok          1.5s   2 inputs
  compose    ok         0.57s   2 services, db first for the dump
  load db    ok          2.2s   db: psql, 0 stderr lines
  ready      ok          4.7s   postgres accepts connections, gitea answers
  PASS  5/5 checks  -  total 10.5s  -  teardown ok
exit=0
```

That is a real Gitea and a real PostgreSQL, seeded, dumped, backed up with restic,
destroyed, and drillback - by the `drillback` binary out of the release archive, running
in this image, driving the host's daemon.

The one thing the run above does *not* prove is the failure mode the same-path rule
exists to prevent, because it satisfies the rule. If you get `RESTORE UNUSABLE` with
every check failing and inputs that look empty, check that your `--workspace` path is
mounted at the same path on both sides before you suspect your backup.

If you run this on Linux and the drill does not work, that is a bug worth reporting:
open an issue with the exact `docker run` you used.
