package pudd

import (
	"context"
	"log"
	"os"
	"os/signal"

	"pudd/internal/config"
	"pudd/internal/gcs"
	"pudd/internal/store"
	"pudd/internal/worker"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	cfg := config.FromFlags()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		logger.Fatalf("Open DB: %v", err)
	}
	defer db.Close()

	if err := store.Init(db); err != nil {
		logger.Fatalf("Init DB: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	gcsClient, err := gcs.NewClient(ctx, cfg)
	if err != nil {
		logger.Fatalf("GCS client: %v", err)
	}
	defer gcsClient.Close()

	uploader := gcs.NewUploader(gcsClient, cfg.Bucket, cfg.ObjectPrefix)

	logger.Printf("pudd starting (db=%s bucket=%s workers=%d)", 
		cfg.DBPath, cfg.Bucket, cfg.Workers,
	)

	worker.Run(ctx, logger, db, cfg, uploader)
}