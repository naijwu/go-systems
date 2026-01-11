package camerautil

import (
	"fmt"
	"os"
	"os/exec"
)

func RemountRW(mountPoint string) error {
	return exec.Command("mount", "-o", "remount,rw", mountPoint).Run()
}

func RemountRO(mountPoint string) error {
	return exec.Command("mount", "-o", "remount,ro", mountPoint).Run()
}

// DeleteFromCamera deletes a file at absolute path under mountPoint.
// Caller should ensure mountPoint is the correct mounted camera filesystem.
func DeleteFromCamera(mountPoint string, absPath string) error {
	// remount RW, delete, sync, remount RO
	if err := RemountRW(mountPoint); err != nil {
		return fmt.Errorf("remount rw: %w", err)
	}
	defer func() { _ = RemountRO(mountPoint) }()

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	// Best-effort sync
	_ = exec.Command("sync").Run()
	return nil
}
