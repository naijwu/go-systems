package ui

import (
	"context"
	"database/sql"
	"sort"
	"sync"

	"pudd/internal/progress"
	"pudd/internal/store"
)

type Phase string

const (
	PhaseIdle      Phase = "idle"
	PhaseUploading Phase = "uploading"
	PhaseDone      Phase = "done"
	PhaseError     Phase = "error"
)

type ActiveUpload struct {
	FileID     int64  `json:"file_id"`
	DeviceID   string `json:"device_id"`
	SrcPath    string `json:"src_path"`
	ObjectName string `json:"object_name"`
	BytesSent  int64  `json:"bytes_sent"`
	TotalBytes int64  `json:"total_bytes"`
	Percent    int64  `json:"percent"`
}

type OverallStatus struct {
	UploadedFiles int64 `json:"uploaded_files"`
	TotalFiles    int64 `json:"total_files"`
	UploadedBytes int64 `json:"uploaded_bytes"`
	TotalBytes    int64 `json:"total_bytes"`
	Percent       int64 `json:"percent"`
}

type StatusSnapshot struct {
	Phase         Phase          `json:"phase"`
	ActiveUploads []ActiveUpload `json:"active_uploads"`
	Overall       OverallStatus  `json:"overall"`
	LastError     string         `json:"last_error"`
}

type StatusService struct {
	db *sql.DB

	mu      sync.RWMutex
	uploads map[int64]progress.UploadUpdate
}

func NewStatusService(db *sql.DB) *StatusService {
	return &StatusService{
		db:      db,
		uploads: make(map[int64]progress.UploadUpdate),
	}
}

func (s *StatusService) PublishUpload(update progress.UploadUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch update.Phase {
	case progress.UploadStarted, progress.UploadProgress:
		s.uploads[update.FileID] = update
	case progress.UploadCompleted, progress.UploadFailed:
		delete(s.uploads, update.FileID)
	}
}

func (s *StatusService) Snapshot(ctx context.Context) (StatusSnapshot, error) {
	summary, err := store.FetchStatusSummary(ctx, s.db)
	if err != nil {
		return StatusSnapshot{}, err
	}

	activeUploads, activeBytes := s.activeUploads()
	uploadedBytes := summary.UploadedBytes + activeBytes
	if uploadedBytes > summary.TotalBytes {
		uploadedBytes = summary.TotalBytes
	}

	snapshot := StatusSnapshot{
		ActiveUploads: activeUploads,
		Overall: OverallStatus{
			UploadedFiles: summary.UploadedFiles,
			TotalFiles:    summary.TotalFiles,
			UploadedBytes: uploadedBytes,
			TotalBytes:    summary.TotalBytes,
			Percent:       percentComplete(uploadedBytes, summary.TotalBytes),
		},
		LastError: summary.LatestError,
	}

	switch {
	case summary.PendingErrors > 0:
		snapshot.Phase = PhaseError
	case summary.TotalFiles == 0:
		snapshot.Phase = PhaseIdle
	case uploadedBytes >= summary.TotalBytes:
		snapshot.Phase = PhaseDone
	default:
		snapshot.Phase = PhaseUploading
	}

	return snapshot, nil
}

func (s *StatusService) activeUploads() ([]ActiveUpload, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ActiveUpload, 0, len(s.uploads))
	var totalBytesSent int64
	for _, update := range s.uploads {
		out = append(out, ActiveUpload{
			FileID:     update.FileID,
			DeviceID:   update.DeviceID,
			SrcPath:    update.SrcPath,
			ObjectName: update.ObjectName,
			BytesSent:  update.BytesSent,
			TotalBytes: update.TotalBytes,
			Percent:    percentComplete(update.BytesSent, update.TotalBytes),
		})
		totalBytesSent += update.BytesSent
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].FileID < out[j].FileID
	})

	return out, totalBytesSent
}

func percentComplete(done, total int64) int64 {
	if total <= 0 {
		return 0
	}
	if done >= total {
		return 100
	}
	return (done * 100) / total
}
