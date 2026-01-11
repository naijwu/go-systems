package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pudd/internal/camerautil"
	"pudd/internal/config"
	"pudd/internal/copyutil"
	"pudd/internal/hash"
	"pudd/internal/model"
	"pudd/internal/store"
)

type Uploader interface {
	UploadAndVerify(ctx context.Context, f model.FileRow) error
}

func Run(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, uploader Uploader) {
	jobs := make(chan model.FileRow, cfg.Workers*2)

	for i := 0; i < cfg.Workers; i++ {
		i := i
		go workerLoop(ctx, logger, db, cfg, uploader, i, jobs)
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(jobs)
			return
		case <-ticker.C:
			rows, err := store.FetchRunnable(db, 100)
			if err != nil {
				logger.Printf("pipeline fetch error: %v", err)
				continue
			}
			for _, f := range rows {
				select {
				case jobs <- f:
				default:
					break
				}
			}
		}
	}
}

func workerLoop(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, uploader Uploader, idx int, jobs <-chan model.FileRow) {
	workerID := "pipe-" + strconvI(idx) + "-" + strconvI(os.Getpid())

	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-jobs:
			if !ok { return }

			switch f.State {
			case model.StateDiscovered:
				handleDiscovered(ctx, logger, db, cfg, workerID, f)
			case model.StateQueued:
				if uploader == nil {
					// Upload not configured
					continue
				}
				handleQueued(ctx, logger, db, cfg, workerID, f, uploader)
			case model.StateVerified:
				handleVerified(ctx, logger, db, cfg, workerID, f)
			default:
				// ignore
			}
		}
	}
}

func handleDiscovered(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, workerID string, f model.FileRow) {
	claimed, err := store.ClaimDiscovered(db, f.ID, workerID, cfg.Lease)
	if err != nil || !claimed {
		return
	}

	// Compute absolute source file path from mount root + device_id + src_path
	srcAbs := filepath.Join(cfg.MountRoot, f.DeviceID, strings.TrimPrefix(f.SrcPath, "/"))

	// Copy with atomic tmp + fsync + rename
	if err := copyutil.CopyAtomic(srcAbs, f.StagedPath); err != nil {
		store.MarkErrorWithBackoff(db, f.ID, err)
		return
	}

	// Optional: delete from camera right after copy (DANGEROUS)
	if cfg.DeleteCameraAfterCopy {
		mountPoint := filepath.Join(cfg.MountRoot, f.DeviceID)
		if err := camerautil.DeleteFromCamera(mountPoint, srcAbs); err != nil {
			// If deletion fails, do NOT fail the pipeline; just log + continue.
			// (You may want a separate "camera_deleted" flag later.)
			logger.Printf("[%s] camera delete failed file=%d: %v", workerID, f.ID, err)
		}
	}

	// Update state to COPIED
	_ = store.Transition(db, f.ID, model.StateCopying, model.StateCopied)

	// Hash the staged file once
	h, err := hash.Compute(f.StagedPath)
	if err != nil {
		store.MarkErrorWithBackoff(db, f.ID, err)
		return
	}
	if err := store.UpdateHashes(db, f.ID, h.Size, h.SHA256, h.CRC32C); err != nil {
		store.MarkErrorWithBackoff(db, f.ID, err)
		return
	}

	_ = store.Transition(db, f.ID, model.StateCopied, model.StateHashed)
	_ = store.Transition(db, f.ID, model.StateHashed, model.StateQueued)
}

func handleQueued(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, workerID string, f model.FileRow, uploader Uploader) {
	claimed, err := store.ClaimQueued(db, f.ID, workerID, cfg.Lease)
	if err != nil || !claimed {
		return
	}

	// Ensure we have hashes (in case you inserted QUEUED elsewhere)
	if f.Size == 0 || f.SHA256 == "" || f.CRC32C == 0 {
		h, err := hash.Compute(f.StagedPath)
		if err != nil {
			store.MarkErrorWithBackoff(db, f.ID, err)
			return
		}
		if err := store.UpdateHashes(db, f.ID, h.Size, h.SHA256, h.CRC32C); err != nil {
			store.MarkErrorWithBackoff(db, f.ID, err)
			return
		}
		f.Size, f.SHA256, f.CRC32C = h.Size, h.SHA256, h.CRC32C
	}

	if err := uploader.UploadAndVerify(ctx, f); err != nil {
		store.MarkErrorWithBackoff(db, f.ID, err)
		return
	}

	_ = store.Transition(db, f.ID, model.StateUploading, model.StateUploaded)
	_ = store.Transition(db, f.ID, model.StateUploaded, model.StateVerified)
}

func handleVerified(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, workerID string, f model.FileRow) {
	claimed, err := store.ClaimVerified(db, f.ID, workerID, cfg.Lease)
	if err != nil || !claimed {
		return
	}

	if cfg.DeleteLocalAfterVerify {
		if err := os.Remove(f.StagedPath); err != nil {
			// keep it retriable
			store.MarkErrorWithBackoff(db, f.ID, err)
			_ = store.Transition(db, f.ID, model.StateCleaning, model.StateVerified)
			return
		}
	}

	_ = store.Transition(db, f.ID, model.StateCleaning, model.StateDone)
}

func strconvI(v int) string {
	return fmt.Sprintf("%d", v)
}
