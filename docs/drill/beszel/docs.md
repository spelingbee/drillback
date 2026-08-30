# Beszel - the official backup documentation

- **Application:** [henrygd/beszel](https://github.com/henrygd/beszel), 24,830 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.18.8 (the hub). Image `henrygd/beszel:0.18.8`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No. <https://beszel.dev/guide/getting-started> links 30-odd guide pages - hub
installation, agent installation, environment variables, notifications, GPU, SMART data,
security, systemd, third-party integrations - and none of them is about backups.
`https://beszel.dev/guide/backups` answers 404.

The application does have a backup feature: PocketBase's, reachable through the
interface. Nothing in the documentation mentions it.

## What the documentation does say

From [hub installation](https://beszel.dev/guide/hub-installation):

> All methods will start the Beszel service on port 8090 and mount the `./beszel_data`
> directory for persistent storage.

```yaml
services:
  beszel:
    image: henrygd/beszel
    container_name: beszel
    restart: unless-stopped
    environment:
      - APP_URL=http://localhost:8090
    ports:
      - 8090:8090
    volumes:
      - ./beszel_data:/beszel_data
```

One directory, named once, described as "persistent storage". That is the whole of it.

From [environment variables](https://beszel.dev/guide/environment-variables), the two
the drill uses to have an account it knows the password of:

> `USER_EMAIL` - Create first user with this email.
> `USER_PASSWORD` - Create first user with this password.

## The two readings

- **A: the data directory** - `/beszel_data`.
- **B: "the database"** - `data.db`, the PocketBase database inside it.

Both were tested. See [result.md](result.md).
