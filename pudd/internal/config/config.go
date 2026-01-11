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

	flag.Parse()

	return cfg
}