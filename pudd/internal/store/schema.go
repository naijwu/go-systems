package store

import "database/sql"


func Init(db *sql.DB) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`PRAGMA foreign_keys=ON;`,
		`
CREATE TABLE IF NOT EXISTS files (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  device_id    TEXT NOT NULL,
  src_path     TEXT NOT NULL,
  staged_path  TEXT NOT NULL,

  size         INTEGER NOT NULL DEFAULT 0,
  sha256       TEXT NOT NULL DEFAULT '',
  crc32c       INTEGER NOT NULL DEFAULT 0,

  state        TEXT NOT NULL,
  attempts     INTEGER NOT NULL DEFAULT 0,
  last_error   TEXT NOT NULL DEFAULT '',
  next_run_at  TEXT,
  claimed_by   TEXT NOT NULL DEFAULT '',
  claim_until  TEXT,
  updated_at   TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),

  UNIQUE(device_id, src_path, size)
);

CREATE INDEX IF NOT EXISTS idx_files_state_next
ON files(state, next_run_at);

CREATE INDEX IF NOT EXISTS idx_files_claim_until
ON files(claim_until);
`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}