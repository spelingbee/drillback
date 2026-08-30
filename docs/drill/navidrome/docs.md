# Navidrome - the official backup documentation

- **Application:** [navidrome/navidrome](https://github.com/navidrome/navidrome),
  23,213 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.63.2. Image `deluan/navidrome:0.63.2`.
- **Documentation read:** 2026-08-30. The page says
  "Last modified January 4, 2026".

## Is there a backup page?

Yes: <https://www.navidrome.org/docs/usage/admin/backup/>, titled *Automated Backup*.
It is the best backup documentation in this drill so far, and it is the reason this
result is worth reading: the page is careful, and the procedure it documents still does
not work.

## What it says

On scope - and this is the sentence other projects should copy:

> Note: The backup process ONLY backs up the database (users, play counts, etc.). It
> does NOT back up the music or the config.

On configuration, both ways:

```toml
[Backup]
Path = "/path/to/backup/folder"
Count = 7
Schedule =  "0 0 * * *"
```

```yaml
environment:
  ND_BACKUP_PATH: /backup
  ND_BACKUP_SCHEDULE: "0 0 * * *"
  ND_BACKUP_COUNT: 7
volumes:
  - ./data:/data
  - ./backup:/backup
```

On taking one by hand:

> You can manually create a backup via the `navidrome backup create` command:
>
> ```sh
> sudo navidrome backup create
> ```
>
> If you use docker compose, you can do the same with:
>
> ```sh
> sudo docker compose run <service_name> backup create
> ```

And on putting one back, in full - this is the entire restore section:

> **Restoring a Backup**
>
> When you restore a backup, the existing data in the database is wiped and the data in
> the backup gets copied into the database.
>
> Note: YOU MUST BE SURE TO RUN THIS COMMAND WHILE THE NAVIDROME APP IS NOT
> RUNNING/LIVE.
>
> Restore a backup by running the `navidrome backup restore` command.
>
> **Attention:** Restoring a backup should ONLY be done when the service is NOT running.
> You've been warned.

Three sentences and two warnings, both about the same thing. What is not there:

- the `--backup-file` flag, which is how you say *which* backup to restore, and the fact
  that its value is resolved inside `Backup.Path` - an absolute path does not work;
- `--force`, without which the command asks a question no automation can answer;
- that the command refuses to run when there is no database yet, which is the state a
  machine is in when you are restoring onto a new one.

## The reading

There is only one: `navidrome backup create`, then `navidrome backup restore`. The drill
tested that, then tested it again with the two undocumented conditions satisfied, and
kept a copy of the whole `/data` folder as a control. See [result.md](result.md).
