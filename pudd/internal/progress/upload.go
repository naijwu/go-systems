package progress

import (
	"log"

	"pudd/internal/model"
)

type UploadPhase string

const (
	UploadStarted   UploadPhase = "started"
	UploadProgress  UploadPhase = "progress"
	UploadCompleted UploadPhase = "completed"
	UploadFailed    UploadPhase = "failed"
)

type UploadUpdate struct {
	Phase      UploadPhase
	FileID     int64
	DeviceID   string
	SrcPath    string
	StagedPath string
	ObjectName string
	BytesSent  int64
	TotalBytes int64
	Error      string
}

type Publisher interface {
	PublishUpload(UploadUpdate)
}

type MultiPublisher struct {
	Publishers []Publisher
}

func (p MultiPublisher) PublishUpload(update UploadUpdate) {
	for _, publisher := range p.Publishers {
		if publisher == nil {
			continue
		}
		publisher.PublishUpload(update)
	}
}

type NopPublisher struct{}

func (NopPublisher) PublishUpload(UploadUpdate) {}

type LoggerPublisher struct {
	Logger *log.Logger
}

func (p LoggerPublisher) PublishUpload(update UploadUpdate) {
	if p.Logger == nil {
		return
	}

	switch update.Phase {
	case UploadStarted:
		p.Logger.Printf("[upload] start file=%d object=%s src=%s", update.FileID, update.ObjectName, update.SrcPath)
	case UploadProgress:
		percent := percentComplete(update.BytesSent, update.TotalBytes)
		p.Logger.Printf("[upload] progress file=%d object=%s sent=%d/%d (%d%%)", update.FileID, update.ObjectName, update.BytesSent, update.TotalBytes, percent)
	case UploadCompleted:
		p.Logger.Printf("[upload] complete file=%d object=%s", update.FileID, update.ObjectName)
	case UploadFailed:
		p.Logger.Printf("[upload] failed file=%d object=%s err=%s", update.FileID, update.ObjectName, update.Error)
	}
}

func UpdateFromFile(phase UploadPhase, f model.FileRow, objectName string) UploadUpdate {
	return UploadUpdate{
		Phase:      phase,
		FileID:     f.ID,
		DeviceID:   f.DeviceID,
		SrcPath:    f.SrcPath,
		StagedPath: f.StagedPath,
		ObjectName: objectName,
		BytesSent:  0,
		TotalBytes: f.Size,
	}
}

func percentComplete(sent, total int64) int64 {
	if total <= 0 {
		return 0
	}
	if sent >= total {
		return 100
	}
	return (sent * 100) / total
}
