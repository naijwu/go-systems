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
  -status-addr 127.0.0.1:8787 \    # local HTTP status endpoint
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
- `ui`: exposes a local HTTP status API for the on-device UI.

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
                                +--------------> [ui /status]
                                                       |
                                                       +--> [copyutil] --> staged file
                                                       +--> [hash]
                                                       +--> [gcs] (upload to cloud)
                                                       +--> [camerautil] (delete on-camera)
                                                       |
                                                       v
                                                  [store / sqlite]
```

## Local Status API

`pudd` exposes a device-local HTTP status endpoint on `127.0.0.1:8787` by default:

```text
GET /status
```

Example response:

```json
{
  "phase": "uploading",
  "active_uploads": [
    {
      "file_id": 12,
      "device_id": "BD5600905",
      "src_path": "/DCIM/MOVIE/20260215174219_000002.MP4",
      "object_name": "devices/dev-test-001/BD5600905/12.bin",
      "bytes_sent": 10485760,
      "total_bytes": 52428800,
      "percent": 20
    }
  ],
  "overall": {
    "uploaded_files": 3,
    "total_files": 10,
    "uploaded_bytes": 167772160,
    "total_bytes": 524288000,
    "percent": 32
  },
  "last_error": ""
}
```

Notes:

- `active_uploads` may contain more than one item when multiple workers are uploading concurrently.
- `overall` is byte-based:
  `uploaded_bytes / total_bytes`
- `phase` is one of `idle`, `uploading`, `done`, or `error`.
- The endpoint is intended for a local UI on the device and binds to loopback by default.

