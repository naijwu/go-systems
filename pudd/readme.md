# `pudd`

From `pudd/`:

```sh
go run ./cmd/pudd
```

Example with local writable paths (from `run-dev.sh`):

```sh
go run ./cmd/pudd \
  -bucket pudd \                   # Google cloud storagte
  -prefix devices/dev-test-001 \   # GCS bucket prefix
  -creds ./etc/pudd-dev-sa.json \  # GCS credentials
  -db ./pudd.db \                  # sqlite database file
  -mount-root ./tmp/mnt \          # final mount directory for devices
  -probe-root ./tmp/mnt/_probe \   # temporary mount used to inspect devices
  -stage-root ./tmp/staging        # local staging area before upload
```

For real device mounts/udev access, run with `sudo`.

## Packages

- `config`: parses CLI flags into runtime config values.
- `model`: defines file rows and upload state-machine states.
- `store`: owns SQLite schema, queries, claims, transitions, and retries.
- `udev`: watches for USB block-device add/remove events.
- `mount`: mounts and unmounts detected devices.
- `deviceid`: derives a stable device ID from camera/media metadata.
- `discover`: scans mounted media and inserts `DISCOVERED` files into SQLite.
- `pipeline`: polls runnable files and advances them through copy, hash, upload, and cleanup.
- `copyutil`: copies files into staging atomically.
- `hash`: computes file size, SHA256, and CRC32C.
- `gcs`: provides the cloud uploader and upload verification.
- `camerautil`: optionally remounts RW to delete files from the camera after copy.

## Runtime

```text
Camera via USB
   |
   v
[udev] ---> [main] ---> [mount] ---> [deviceid] ---> [discover]
                                |                      |
                                |                      v
                                |                 [store / sqlite]
                                |                      |
                                |                      v
                                +--------------> [pipeline]
                                                       |
                                                       +--> [copyutil] --> staged file
                                                       +--> [hash]
                                                       +--> [gcs] (upload to cloud)
                                                       +--> [camerautil] (delete on-camera)
                                                       |
                                                       v
                                                  [store / sqlite]
```

