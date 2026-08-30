# Open WebUI - the official backup documentation

- **Application:** [open-webui/open-webui](https://github.com/open-webui/open-webui),
  150,348 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v0.11.1, the current release on the day of the drill. Image
  `ghcr.io/open-webui/open-webui:v0.11.1` (7.16 GB).
- **Documentation read:** 2026-08-30.

## Is there a backup page?

Yes, and it is a good one: <https://docs.openwebui.com/tutorials/maintenance/backups>.
Open WebUI is the first application in this drill with a page whose subject is backups.

## What it says

On why:

> "Docker containers are ephemeral and data must be persisted to ensure its survival on
> the host filesystem."

On where the data is - the standard deployment:

```yaml
open-webui:
  volumes:
    - open-webui:/app/backend/data
```

and the bind-mount alternative:

```yaml
volumes:
  - /opt/ollama:/root/.ollama
  - /opt/open-webui:/app/backend/data
```

On what is in it - the page lists five things by name:

| | |
|---|---|
| `webui.db` | the SQLite database |
| `vector_db/` | the ChromaDB vector database |
| `uploads/` | user-uploaded files |
| `cache/` | cached data |
| `audit.log` | the audit event log |

On how, with worked commands:

```bash
rsync -av --delete --link-dest="$SNAPSHOT_DIR/$(ls -t "$SNAPSHOT_DIR" | head -n 1)" "$SOURCE_DIR/" "$SNAPSHOT_PATH"
tar -czvf "$CHROMADB_BACKUP_FILE" -C "$SOURCE" vector_db
sqlite3 "$SQLITE_DB_FILE" ".backup '$SQLITE_BACKUP_FILE'"
rclone sync "$SOURCE" "$DEST" $EXCLUDE_ARGS --progress --transfers=32 --checkers=16 --profile "$B2_PROFILE"
```

and it tells you to stop the stack first:

```bash
docker-compose -f "$COMPOSE_FILE" down
```

then bring it back up afterwards.

That is more, and better, than most projects in this drill have. Two things are missing
and both showed up in the run:

- **there is no restore procedure** - the page covers taking the backup and stopping
  there;
- **`cache/` is listed as something to back up**, without a word about what is in it.

## The reading

There is only one honest primary reading: copy `/app/backend/data`, all of it, with the
stack stopped. That is reading A. Reading B is the same directory with `cache/` left
out, which is a deviation from the documented list and is labelled as one - the drill
tested it because reading A could not be restored on this host, and because the numbers
in [result.md](result.md) make it the reading the page should probably recommend.
