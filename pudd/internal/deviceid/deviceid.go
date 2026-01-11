package deviceid

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

type Source string

const (
	SourcePudd Source = "pudd"
	SourceFSUUID Source = "fs_uuid"
	SourceSerialShort Source = "serial_short"
	SourceSerial Source = "serial"
	SourceDevPath Source = "devpath_hash"
)

// Derive picks a stable device id using:
// 1) DCIM/.pudd (authoritative if present)
// 2) ID_FS_UUID
// 3) ID_SERIAL_SHORT
// 4) ID_SERIAL
// 5) sha1(DEVPATH)
func Derive(mountPoint string, udevProps map[string]string) (deviceID string, source Source) {
	if id, ok := readPuddID(mountPoint); ok {
		return sanitize(id), SourcePudd
	}

	if v := udevProps["ID_FS_UUID"]; v != "" {
		return sanitize(v), SourceFSUUID
	}
	if v := udevProps["ID_SERIAL_SHORT"]; v != "" {
		return sanitize(v), SourceSerialShort
	}
	if v := udevProps["ID_SERIAL"]; v != "" {
		return sanitize(v), SourceSerial
	}

	// last resort: hash devpath (stable-ish on same host, not great across re-enumerations)
	h := sha1.Sum([]byte(udevProps["DEVPATH"]))
	return "usb-" + hex.EncodeToString(h[:8]), SourceDevPath
}

func readPuddID(mountPoint string) (string, bool) {
	// Primary location: DCIM/.pudd
	paths := []string{
		filepath.Join(mountPoint, "DCIM", ".pudd"),
		// optional fallbacks if you want:
		// filepath.Join(mountPoint, ".pudd"),
	}

	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "pudd_id=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "pudd_id=")), true
			}
			// allow bare id file (single line)
			if !strings.Contains(line, "=") {
				return line, true
			}
		}
	}
	return "", false
}

func sanitize(s string) string {
	// Keep it path-safe for mount folders and cloud prefixes.
	// Replace whitespace with underscores; remove path separators.
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}
