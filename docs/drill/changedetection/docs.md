# changedetection.io - the official backup documentation

- **Application:** [dgtlmoon/changedetection.io](https://github.com/dgtlmoon/changedetection.io),
  33,415 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.55.8. Image `dgtlmoon/changedetection.io:0.55.8`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

There is a backup *feature* with a page in the application itself - **Backups**, with a
**Create backup** button and a **Restore** tab - and a wiki page about putting a backup
back: [Restoring backup files](https://github.com/dgtlmoon/changedetection.io/wiki/Restoring-backup-files).

There is no page in the documentation that says *when* or *why* to take one, which is
the ordinary state of affairs in this drill and not a complaint.

## What the wiki page says

Quoted in full, because it is short and because it gets the important part right:

> Important first is to stop the changedetection.io instance, as it will be monitoring
> and writing to the disk.
>
>     docker kill changedetection
>
> Change the path in `BACKUP_PATH` to be the FULL path to the export backup Zip file
>
> ```
> export BACKUP_PATH=/home/dgtlmoon/Downloads/changedetection-backup-1616751572.zip
>
> docker run -v changedetectionio_changedetection-data:/datastore -v $BACKUP_PATH:/backup.zip python:3.8-slim bash -c "apt-get update && apt-get install unzip -y; unzip -o /backup.zip -d /datastore"
> ```
>
> then restart
>
>     docker restart changedetection

"Important first is to stop the instance, as it will be monitoring and writing to the
disk" is the sentence most of the applications in this drill are missing.

## What the README says about the data

```sh
docker run -d --restart always -p "127.0.0.1:5000:5000" -v datastore-volume:/datastore --name changedetection.io dgtlmoon/changedetection.io
```

One volume, `/datastore`. Inside it: `changedetection.json` with every watch's
configuration, `secret.txt`, and one directory per watch holding that watch's history -
the text snapshots, the compressed HTML, and `history.txt`.

## The reading

One: the zip the Backups page writes, restored by unzipping it into `/datastore`. A copy
of `/datastore` was taken as a control. See [result.md](result.md).
