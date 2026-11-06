// Package libdocker inspects Docker Desktop and runs short-lived containers.
package libdocker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrDesktopNotRunning is returned when the Docker Desktop app itself is not
// running, as opposed to its VM being stopped or paused.
var ErrDesktopNotRunning = errors.New("docker desktop is not running")

// DesktopStatus is the output of `docker desktop status --format json`.
type DesktopStatus struct {
	Status    string // e.g. "running", "stopped"
	SessionID string
}

// DockerDesktopStatus asks the docker CLI for Docker Desktop's state. Unlike
// most docker commands it does not wake the VM from Resource Saver mode.
func DockerDesktopStatus(ctx context.Context) (DesktopStatus, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "desktop", "status", "--format", "json")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return DesktopStatus{}, statusError(err, stderr.String())
	}
	return parseDesktopStatus(out)
}

func statusError(err error, stderr string) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("docker desktop status: %w", err)
	}
	if strings.Contains(stderr, "Could not retrieve status") {
		return ErrDesktopNotRunning
	}
	return fmt.Errorf("docker desktop status: %w: %s", err, strings.TrimSpace(stderr))
}

func parseDesktopStatus(out []byte) (DesktopStatus, error) {
	var s DesktopStatus
	if err := json.Unmarshal(out, &s); err != nil {
		return DesktopStatus{}, fmt.Errorf("parsing docker desktop status: %w", err)
	}
	return s, nil
}
