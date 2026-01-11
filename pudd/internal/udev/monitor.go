package udev

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

type Event struct {
	Action string // add/remove
	DevName string // /dev/sda1
	DevPath string // DEVPATH=...
	Props map[string]string // key=value from udev
}

// Run listens to udev block events and calls onEvent for USB partitions only.
func Run(ctx context.Context, onEvent func(Event)) error {
	cmd := exec.Command(
		"udevadm",
		"monitor",
		"--udev",
		"--subsystem-match=block",
		"--property",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// Ensure subprocess is killed on context cancellation.
	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()

	sc := bufio.NewScanner(stdout)
	props := map[string]string{}

	flush := func() {
		if len(props) == 0 {
			return
		}
		// Filter: USB partition add/remove only.
		if props["ID_BUS"] != "usb" {
			props = map[string]string{}
			return
		}
		if props["DEVTYPE"] != "partition" {
			props = map[string]string{}
			return
		}
		action := props["ACTION"]
		if action != "add" && action != "remove" {
			props = map[string]string{}
			return
		}

		onEvent(Event{
			Action:  action,
			DevName: props["DEVNAME"],
			DevPath: props["DEVPATH"],
			Props:   props,
		})
		props = map[string]string{}
	}

	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(sc.Text())
		if line == "" {
			flush()
			continue
		}
		if strings.Contains(line, "=") {
			kv := strings.SplitN(line, "=", 2)
			props[kv[0]] = kv[1]
		}
	}

	// Scanner ended; try one last flush.
	flush()
	return sc.Err()
}
