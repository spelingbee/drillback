# ConvertX - the official backup documentation

- **Application:** [C4illin/ConvertX](https://github.com/C4illin/ConvertX), 18,630 stars
  in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v0.18.0. Image `ghcr.io/c4illin/convertx:v0.18.0`.
- **Documentation read:** 2026-08-30, from the repository README, which is the
  documentation.

## Is there a backup page?

No, and the word does not appear. The README covers deployment, a table of environment
variables, the supported converters and how to build it. There is nothing about keeping
or restoring anything.

## What the documentation does say

The Deployment section, verbatim:

```yml
# docker-compose.yml
services:
  convertx:
    image: ghcr.io/c4illin/convertx
    container_name: convertx
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - JWT_SECRET=aLongAndSecretStringUsedToSignTheJSONWebToken1234 # will use randomUUID() if unset
      # - HTTP_ALLOWED=true # uncomment this if accessing it over a non-https connection
    volumes:
      - ./data:/app/data
```

> Then visit `http://localhost:3000` in your browser and create your account. Don't
> leave it unconfigured and open, as anyone can register the first account.

> If you get unable to open database file run `chown -R $USER:$USER path` on the path
> you choose.

One volume, `./data:/app/data`. Inside it, on a running instance: `mydb.sqlite` with the
accounts and the conversion jobs, its `-wal` and `-shm`, and two directories of files
those jobs point at - `uploads/<user>/<job>/` and `output/<user>/<job>/`.

One line in the environment table is worth reading twice for a backup:

> `JWT_SECRET` - when unset it will use the value from `randomUUID()`

It signs session tokens and encrypts nothing in the database, so a restore does not need
the original. But if it is set, it is in your compose file and not in your backup; and if
it is unset, everybody is signed out on every restart, restore or not.

## The two readings

- **A: the data directory** - `/app/data`, the one thing the README mounts.
- **B: "the database"** - `mydb.sqlite`, the file inside it that holds the accounts.
