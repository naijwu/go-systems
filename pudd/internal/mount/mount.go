package mount

import (
	"os"
	"os/exec"
)

func MountRO(devNode, mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0o755); err != nil {
		return err
	}
	return exec.Command("mount", "-o", "ro", devNode, mountPoint).Run()
}

func Unmount(mountPoint string) error {
	// umount is fine even if already unmounted; caller can ignore error if desired.
	return exec.Command("umount", mountPoint).Run()
}
