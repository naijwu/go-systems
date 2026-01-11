package model

// This package models a file in the sqlite database

type FileState string

const (
	StateQueued FileState = "QUEUED"
	StateUploading FileState = "UPLOADING"
	StateUploaded FileState = "UPLOADED"
	StateVerified FileState = "VERIFIED"
	StateDone FileState = "DONE"
	StateError FileState = "ERROR"
)

type FileRow struct {
	ID int64
	DeviceID string
	SrcPath string
	StagedPath string

	Size int64
	SHA256 string
	CRC32C uint32
	State FileState // to model state machine
	Attempts int64
	LastError string
}