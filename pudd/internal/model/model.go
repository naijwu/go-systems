package model

// This package models a file in the sqlite database (state machine source of truth)

type FileState string

const (
	StateDiscovered FileState = "DISCOVERED"
	StateCopying FileState = "COPYING"
	StateCopied FileState = "COPIED"
	StateHashed FileState = "HASHED"
	StateQueued FileState = "QUEUED"

	StateUploading FileState = "UPLOADING"
	StateUploaded FileState = "UPLOADED"
	StateVerified FileState = "VERIFIED"

	StateCleaning FileState = "CLEANING"
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