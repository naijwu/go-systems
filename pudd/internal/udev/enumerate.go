package udev

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// EnumerateCurrent returns currently attached USB partitions as synthetic add events.
func EnumerateCurrent(ctx context.Context) ([]Event, error) {
	entries, err := os.ReadDir("/sys/class/block")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	var out []Event
	for _, name := range names {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Partitions expose a "partition" file in sysfs.
		if _, err := os.Stat(filepath.Join("/sys/class/block", name, "partition")); err != nil {
			continue
		}

		props, err := deviceProperties(ctx, filepath.Join("/dev", name))
		if err != nil {
			return nil, err
		}

		ev, ok := currentEventFromProps(props)
		if !ok {
			continue
		}
		out = append(out, ev)
	}

	return out, nil
}

func deviceProperties(ctx context.Context, devName string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "udevadm", "info", "--query=property", "--name", devName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	props := map[string]string{}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		props[kv[0]] = kv[1]
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return props, nil
}

func currentEventFromProps(props map[string]string) (Event, bool) {
	if !isUSBPartition(props) {
		return Event{}, false
	}
	return Event{
		Action:  "add",
		DevName: props["DEVNAME"],
		DevPath: props["DEVPATH"],
		Props:   cloneProps(props),
	}, true
}

func eventFromMonitorProps(props map[string]string) (Event, bool) {
	if !isUSBPartition(props) {
		return Event{}, false
	}
	action := props["ACTION"]
	if action != "add" && action != "remove" {
		return Event{}, false
	}
	return Event{
		Action:  action,
		DevName: props["DEVNAME"],
		DevPath: props["DEVPATH"],
		Props:   cloneProps(props),
	}, true
}

func isUSBPartition(props map[string]string) bool {
	return props["ID_BUS"] == "usb" && props["DEVTYPE"] == "partition"
}

func cloneProps(props map[string]string) map[string]string {
	out := make(map[string]string, len(props))
	for k, v := range props {
		out[k] = v
	}
	return out
}
