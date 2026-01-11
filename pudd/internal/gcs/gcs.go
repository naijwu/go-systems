package gcs

import (
	"context"
	"errors"

	"pudd/internal/config"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func NewClient(ctx context.Context, cfg config.Config) (*storage.Client, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("missing -bucket")
	}
	if cfg.CredsJSON != "" {
		return storage.NewClient(ctx, option.WithCredentialsFile(cfg.CredsJSON))
	}
	// fallback to Application Default Credentials 
	return storage.NewClient(ctx)
}
