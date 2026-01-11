package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"pudd/internal/config"
	"pudd/internal/model"
	"pudd/internal/store"
)

func Run(ctx context.Context, logger *log.Logger, db *sql.DB, cfg config.Config, uploader Uploader) {
	jobs := make(chan model.FileRow)

	// workers
	for i := 0; i < cfg.Workers; i++ {
		i := i
		go func() {
			workerID := workerID(i)
			runWorker(ctx, logger, db, cfg, uploader, workerID, jobs)
		}()
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <- ctx.Done():
			close(jobs)
			return
		case <- ticker.C:
			rows, err := store.FetchRunnableQueued(db, 50)
			if err != nil {
				logger.Printf("Scheduler fetch error: %v", err)
				continue
			}
			for _, f := range rows {
				select {
				case jobs <- f:
				default:
					// when workers are busy
					break
				}
			}
		}
	}
}