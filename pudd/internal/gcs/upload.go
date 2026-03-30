package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"pudd/internal/model"
	"pudd/internal/progress"

	"cloud.google.com/go/storage"
)

type Uploader struct {
	client   *storage.Client
	bucket   string
	prefix   string
	progress progress.Publisher
}

func NewUploader(client *storage.Client, bucket, prefix string, publisher progress.Publisher) *Uploader {
	if publisher == nil {
		publisher = progress.NopPublisher{}
	}
	return &Uploader{client: client, bucket: bucket, prefix: prefix, progress: publisher}
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

	w.Metadata = map[string]string{
		"device_id": f.DeviceID,
		"src_path":  f.SrcPath,
		"sha256":    f.SHA256,
	}

	u.progress.PublishUpload(progress.UpdateFromFile(progress.UploadStarted, f, objName))

	// upload
	if _, err := file.Seek(0, 0); err != nil {
		_ = w.Close()
		u.publishFailure(f, objName, err)
		return err
	}

	pr := newProgressReader(file, f, objName, u.progress)
	if _, err := io.Copy(w, pr); err != nil {
		_ = w.Close()
		u.publishFailure(f, objName, err)
		return err
	}
	if err := w.Close(); err != nil {
		u.publishFailure(f, objName, err)
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
		u.publishFailure(f, objName, err)
		return err
	}

	if attrs.Size != f.Size {
		err := fmt.Errorf("verify size mismatch: local=%d remote=%d", f.Size, attrs.Size)
		u.publishFailure(f, objName, err)
		return err
	}
	if attrs.CRC32C != f.CRC32C {
		err := fmt.Errorf("verify crc32c mismatch: local=%d remote=%d", f.CRC32C, attrs.CRC32C)
		u.publishFailure(f, objName, err)
		return err
	}

	done := progress.UpdateFromFile(progress.UploadCompleted, f, objName)
	done.BytesSent = f.Size
	u.progress.PublishUpload(done)

	return nil
}

func (u *Uploader) publishFailure(f model.FileRow, objName string, err error) {
	update := progress.UpdateFromFile(progress.UploadFailed, f, objName)
	update.Error = err.Error()
	u.progress.PublishUpload(update)
}

type progressReader struct {
	reader          io.Reader
	update          progress.UploadUpdate
	publisher       progress.Publisher
	bytesSent       int64
	nextPercentMark int64
	lastEmitAt      time.Time
}

func newProgressReader(reader io.Reader, f model.FileRow, objName string, publisher progress.Publisher) *progressReader {
	return &progressReader{
		reader:          reader,
		update:          progress.UpdateFromFile(progress.UploadProgress, f, objName),
		publisher:       publisher,
		nextPercentMark: 10,
		lastEmitAt:      time.Now(),
	}
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.bytesSent += int64(n)
		r.emitIfNeeded(false)
	}
	if err == io.EOF {
		r.emitIfNeeded(true)
	}
	return n, err
}

func (r *progressReader) emitIfNeeded(force bool) {
	now := time.Now()
	total := r.update.TotalBytes
	shouldEmit := force || now.Sub(r.lastEmitAt) >= 2*time.Second

	if !shouldEmit && total > 0 {
		percent := (r.bytesSent * 100) / total
		shouldEmit = percent >= r.nextPercentMark
	}
	if !shouldEmit {
		return
	}

	r.update.BytesSent = r.bytesSent
	r.publisher.PublishUpload(r.update)
	r.lastEmitAt = now

	if total <= 0 {
		return
	}

	percent := (r.bytesSent * 100) / total
	for r.nextPercentMark <= 100 && percent >= r.nextPercentMark {
		r.nextPercentMark += 10
	}
}
