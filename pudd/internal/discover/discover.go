package discover

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"pudd/internal/model"
	"pudd/internal/store"
)

// DiscoverAndInsert scans known media directories and inserts DISCOVERED rows
func DiscoverAndInsert(ctx context.Context, db *sql.DB, deviceID, mountPoint, stageRoot string) error {
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return err
	}

	var roots []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(name, "Movies") || strings.EqualFold(name, "DCIM") {
			roots = append(roots, filepath.Join(mountPoint, name))
		}
	}

	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			continue
		}

		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if d.IsDir() {
				return nil
			}

			// files are in mp4
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".mp4") {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(mountPoint, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)

			srcRel := "/" + rel
			stagedPath := filepath.Join(stageRoot, deviceID, filepath.FromSlash(rel))

			row := store.DiscoveredRow{
				DeviceID: deviceID,
				SrcPath:srcRel,
				StagedPath: stagedPath,
				Size: info.Size(),
				State: model.StateDiscovered,
			}
			return store.InsertDiscovered(db, row)
		})

		if err != nil {
			return err
		}
	}

	return nil
}
