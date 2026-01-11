package main

import (
	"bufio" // buffered IO
	"fmt"   // formatted IO
	"log"
	"os/exec"
	"strings"
)

// simple daemon that just listens to UART changes

func main() {
	log.Println("daemon listening...")

	cmd := exec.Command(
		"udevadm",
		"monitor",
		"--udev",
		"--subsystem-match=block",
		"--property",
	)	

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get StdoutPipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start udevadm: %v", err)
	}

	scanner := bufio.NewScanner(stdout)

	event := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// event termination -- handle and flush
		if line == "" {
			handleEvent(event)
			event = make(map[string]string)
			continue
		}

		// key value lines -- build up
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			event[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Scanner error: %v", err)
	}
}

func handleEvent(ev map[string]string) {
	if ev["ID_BUS"] != "usb" {
		return
	}

	action := ev["ACTION"]
	dev := ev["DEVNAME"]
	serial := ev["ID_SERIAL_SHORT"]
	if serial == "" {
		serial = ev["ID_SERIAL"]
	}

	fmt.Printf(
		"[USB %s] device=%s serial=%s\n",
		action,
		dev,
		serial,
	)
}