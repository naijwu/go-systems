# `pudd`

From `pudd/`:

```sh
go run ./cmd/pudd
```

Example with local writable paths:

```sh
go run ./cmd/pudd \
  -db ./pudd.db \                  # sqlite database file
  -mount-root ./tmp/mnt \          # final mount directory for devices
  -probe-root ./tmp/mnt/_probe \   # temporary mount used to inspect devices
  -stage-root ./tmp/staging        # local staging area before upload
```

If you need real device mounts/udev access, run the same command with `sudo`.
