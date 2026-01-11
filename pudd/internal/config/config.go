package config

import (
	"flag"
	"time"
)

type Config struct {
	DBPath string
	Workers int
	PollInterval time.Duration
	Lease time.Duration

	// GCS
	Bucket string
	ObjectPrefix string
	CredsJSON string

	// Serial/device
	MountRoot string
	ProbeRoot string
	StageRoot string

	// File management behavior
	DeleteCameraAfterCopy bool
	DeleteLocalAfterVerify bool
}

func FromFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.DBPath, "db", "./pudd.db", "path to sqlite DB")
	flag.IntVar(&cfg.Workers, "workers", 2, "number of upload workers")
	flag.DurationVar(&cfg.PollInterval, "poll", 750 * time.Millisecond, "scheduler poll interval")
	flag.DurationVar(&cfg.Lease, "lease", 2 * time.Minute, "upload lease duration")

	flag.StringVar(&cfg.Bucket, "bucket", "", "GCS bucket name")
	flag.StringVar(&cfg.ObjectPrefix, "prefix", "pudd", "GCS object key prefix")
	flag.StringVar(&cfg.CredsJSON, "creds", "", "path to service account JSON")

	flag.StringVar(&cfg.MountRoot, "mount-root", "/mnt/dock", "mount root for cameras")
	flag.StringVar(&cfg.ProbeRoot, "probe-root", "/mnt/dock/_probe", "temporary probe mounts")
	flag.StringVar(&cfg.StageRoot, "stage-root", "/var/lib/pudd/staging", "staging root on SSD")

	flag.BoolVar(&cfg.DeleteCameraAfterCopy, "delete-camera-after-copy", false, "DANGEROUS: delete camera file after successful copy (requires RW remount)")
	flag.BoolVar(&cfg.DeleteLocalAfterVerify, "delete-local-after-verify", true, "delete staged file after GCS verify")

	flag.Parse()
	return cfg
}