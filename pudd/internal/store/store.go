package store

import (
	"database/sql"
	"fmt"
	"time"

	"pudd/internal/model"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

func FetchRunnableQueued(db *sql.DB, limit int) ([]model.FileRow, error) {
	rows, err := db.Query(`
SELECT id, device_id, src_path, staged_path, size, sha256, crc32c, state, attempts, last_error
FROM files
WHERE state = ? AND (next_run_at IS NULL OR next_run_at <= CURRENT_TIMESTAMP)
ORDER BY id
LIMIT ?
`, string(model.StateQueued), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.FileRow
	for rows.Next() {
		var f model.FileRow
		var stateStr string
		var crc32c int64
		if err := rows.Scan(
			&f.ID, &f.DeviceID, &f.SrcPath, &f.StagedPath, &f.Size, &f.SHA256, &crc32c, &stateStr, &f.Attempts, &f.LastError,
		); err != nil {
			return nil, err
		}
		f.CRC32C = uint32(crc32c)
		f.State = model.FileState(stateStr)
		out = append(out, f)
	}
	
	return out, rows.Err()
}

// claim file for upload with lease
func ClaimForUpload(db *sql.DB, fileID int64, claimedBy string, lease time.Duration) (bool, error) {
	res, err := db.Exec(`
UPDATE files
SET state = ?, claimed_by = ?, claim_until = datetime('now', ?), updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND (state = ? OR (state = ? AND (claim_until IS NULL OR claim_until < CURRENT_TIMESTAMP)))	
`, string(model.StateUploading), claimedBy, sqliteDuration(lease), fileID, string(model.StateQueued), string(model.StateUploading),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// transition a file to a state
func Transition(db *sql.DB, fileID int64, from, to model.FileState) error {
	res, err := db.Exec(`
UPDATE files
SET state = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND state = ?
`, string (to), fileID, string(from))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("Transition %s -> %s failed for file=%d", from, to, fileID)
	}
	return nil;
}

// for updating hashes post network action
func UpdateHashes(db *sql.DB, fileID int64, size int64, sha256 string, crc32c uint32) error {
	_, err := db.Exec(`
UPDATE files
SET size=?, sha256=?, crc32c=?, updated_at=CURRENT_TIMESTAMP
WHERE id=?`,
		size, sha256, int64(crc32c), fileID,
	)
	return err
}

func MarkErrorWithBackoff(db *sql.DB, fileID int64, cause error) {
	var attempts int64
	_ = db.QueryRow(`SELECT attempts FROM files WHERE id=?`, fileID).Scan(&attempts)
	attempts++

	// exponential backoff
	delay := time.Second * time.Duration(1 << min64(attempts, 10))
	nextRun := time.Now().Add(delay).UTC().Format("2006-01-02 16:01:02")

	msg := cause.Error()
	if len(msg) > 500 {
		msg = msg[:500]
	}

	_, _ = db.Exec(`
UPDATE files
SET state = ?, attempts = ?, last_error = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE id=?`, 
		string(model.StateError), attempts, msg, nextRun, fileID,
	)

	_, _ = db.Exec(`
UPDATE files
SET state=?, updated_at=CURRENT_TIMESTAMP
WHERE id=? AND state=?`,
		string(model.StateQueued), fileID, string(model.StateError),
	)
}

// utility to convert time.Duration into sql friendly format
func sqliteDuration(d time.Duration) string {
	secs := int(d.Seconds())
	if secs%60 == 0 {
		return fmt.Sprintf("+%d minutes", secs/60)
	}
	return fmt.Sprintf("+%d seconds", secs)
}

// util for getting the mean of two int64
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}