package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"pudd/internal/config"
	"pudd/internal/hash"
	"pudd/internal/model"
	"pudd/internal/store"
)

type Uploader interface {
	UploadAndVerify(ctx context.Context, f model.FileRow) error
}

func workerID(i int) string {
	return fmt.Sprintf("pudd-%d-%d", os.Getpid(), i)
}

func runWorker(
	ctx context.Context,
	logger *log.Logger,
	db *sql.DB,
	cfg config.Config,
	uploader Uploader,
	id string,
	jobs <- chan model.FileRow,
) {
	for {
		select {
		case <- ctx.Done():
			return
		case f, ok := <- jobs:
			if !ok {
				return
			}

			claimed, err := store.ClaimForUpload(db, f.ID, id, cfg.Lease)
			if err != nil {
				logger.Printf("[%s] claim error file=%d: %v", id, f.ID, err)
				continue
			}
			if !claimed {
				continue
			}

			// compute hashes
			if f.Size == 0 || f.SHA256 == "" || f.CRC32C == 0 {
				h, err := hash.Compute(f.StagedPath)
				if err != nil {
					logger.Printf("[%s] hash error file=%d: %v", id, f.ID, err)
					store.MarkErrorWithBackoff(db, f.ID, err)
					continue
				}
				if err := store.UpdateHashes(db, f.ID, h.Size, h.SHA256, h.CRC32C); err != nil {
					logger.Printf("[%s] update hash db error file=%d: %v", id, f.ID, err)
					store.MarkErrorWithBackoff(db, f.ID, err)
					continue
				}

				f.Size = h.Size
				f.SHA256 = h.SHA256
				f.CRC32C = h.CRC32C
			}

			start := time.Now()
			logger.Printf("[%s] uploading file=%d path=%s", id, f.ID, f.StagedPath)

			if err := uploader.UploadAndVerify(ctx, f); err != nil {
				logger.Printf("[%s] upload/verify failed for file=%d: %v", id, f.ID, err)
				store.MarkErrorWithBackoff(db, f.ID, err)
				continue
			}

			// transition states
			
			_ = store.Transition(db, f.ID, model.StateUploading, model.StateUploaded)
			_ = store.Transition(db, f.ID, model.StateUploaded, model.StateVerified)
			_ = store.Transition(db, f.ID, model.StateVerified, model.StateDone)

			logger.Printf("[%s] DONE file=%d dur=%s", id, f.ID, time.Since(start))
		}
	}
}