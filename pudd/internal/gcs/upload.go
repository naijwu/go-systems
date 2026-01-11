package gcs

import (
	"context"
	"fmt"
	"os"
	"time"

	"pudd/internal/model"

	"cloud.google.com/go/storage"
)

type Uploader struct {
	client *storage.Client
	bucket string
	prefix string
}

func NewUploader(client *storage.Client, bucket, prefix string) *Uploader {
	return &Uploader{client: client, bucket: bucket, prefix: prefix}
}

func (u *Uploader) ObjectName(f model.FileRow) string {
	return fmt.Sprintf("%s/%s/%d.bin", u.prefix, f.DeviceID, f.ID)
}

func (u *Uploader) UploadAndVerify(ctx context.Context, f model.FileRow) error {
	objName := u.ObjectName(f)
	bkt := u.client.Bucket(u.bucket)
	obj := bkt.Object(objName)

	file, err := os.Open(f.StagedPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	w := obj.NewWriter(ctx)
	w.ChunkSize = 0
	w.ContentType = "application/octet-stream"

	w.Metadata = map[string]string {
		"device_id": f.DeviceID,
		"src_path": f.SrcPath,
		"sha256": f.SHA256,
	}

	// upload
	if _, err := file.Seek(0, 0); err != nil {
		_ = w.Close()
		return err
	}
	if _, err := file.WriteTo(w); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// fetch attributes and verify
	var attrs *storage.ObjectAttrs
	for i := 0; i < 3; i++ {
		attrs, err = obj.Attrs(ctx)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	
	if attrs.Size != f.Size {
		return fmt.Errorf("verify size mismatch: local=%d remote=%d", f.Size, attrs.Size)
	}
	if attrs.CRC32C != f.CRC32C {
		return fmt.Errorf("verify crc32c mismatch: local=%d remote=%d", f.CRC32C, attrs.CRC32C)
	}
	
	return nil
}