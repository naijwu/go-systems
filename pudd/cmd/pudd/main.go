package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"pudd/internal/config"
	"pudd/internal/deviceid"
	"pudd/internal/discover"
	"pudd/internal/mount"
	"pudd/internal/pipeline"
	"pudd/internal/store"
	"pudd/internal/udev"
	"pudd/internal/worker"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	cfg := config.FromFlags()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		logger.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := store.Init(db); err != nil {
		logger.Fatalf("init db: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start pipeline
	var uploader worker.Uploader
	go pipeline.Run(ctx, logger, db, cfg, uploader)

	// Map devnode -> mountpoint (so remove can unmount the right path)
	var mu sync.Mutex
	devToMount := map[string]string{}

	_ = os.MkdirAll(cfg.ProbeRoot, 0o755)
	_ = os.MkdirAll(cfg.MountRoot, 0o755)

	go func() {
		err := udev.Run(ctx, func(ev udev.Event) {
			switch ev.Action {
			case "add":
				handleAdd(ctx, logger, db, cfg, &mu, devToMount, ev)
			case "remove":
				handleRemove(logger, &mu, devToMount, ev)
			}
		})
		if err != nil && err != context.Canceled {
			logger.Printf("udev monitor error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Println("pudd exiting")
}

func handleAdd(
	ctx context.Context,
	logger *log.Logger,
	db *sql.DB,
	cfg config.Config,
	mu *sync.Mutex,
	devToMount map[string]string,
	ev udev.Event,
) {
	if ev.DevName == "" {
		return
	}

	// 1) Mount to a probe location first (so we can read .pudd)
	probeMP := filepath.Join(cfg.ProbeRoot, filepath.Base(ev.DevName))
	_ = mount.Unmount(probeMP) // ignore errors
	if err := mount.MountRO(ev.DevName, probeMP); err != nil {
		logger.Printf("[add] mount probe failed dev=%s: %v", ev.DevName, err)
		return
	}

	// 2) Derive final device_id (prefers pudd file if present)
	devID, src := deviceid.Derive(probeMP, ev.Props)
	finalMP := filepath.Join(cfg.MountRoot, devID)

	// 3) If probe mountpoint isn't the final desired mountpoint, remount.
	if probeMP != finalMP {
		_ = mount.Unmount(probeMP)
		_ = os.MkdirAll(finalMP, 0o755)
		_ = mount.Unmount(finalMP)
		if err := mount.MountRO(ev.DevName, finalMP); err != nil {
			logger.Printf("[add] mount final failed dev=%s id=%s: %v", ev.DevName, devID, err)
			return
		}
	} else {
		finalMP = probeMP
	}

	mu.Lock()
	devToMount[ev.DevName] = finalMP
	mu.Unlock()

	logger.Printf("[add] dev=%s device_id=%s (source=%s) mount=%s", ev.DevName, devID, src, finalMP)

	// 4) Discover files and insert DISCOVERED rows (idempotent)
	if err := discover.DiscoverAndInsert(ctx, db, devID, finalMP, cfg.StageRoot); err != nil {
		logger.Printf("[add] discover failed id=%s: %v", devID, err)
		return
	}
	logger.Printf("[add] discover complete id=%s", devID)
}

func handleRemove(
	logger *log.Logger,
	mu *sync.Mutex,
	devToMount map[string]string,
	ev udev.Event,
) {
	if ev.DevName == "" {
		return
	}
	mu.Lock()
	mp := devToMount[ev.DevName]
	delete(devToMount, ev.DevName)
	mu.Unlock()

	if mp == "" {
		return
	}
	_ = mount.Unmount(mp)
	logger.Printf("[remove] dev=%s unmounted=%s", ev.DevName, mp)
}
