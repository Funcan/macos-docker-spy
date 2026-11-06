package libdocker

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// Matches the image cmd/probe uses, so the tests don't pull a second one.
const testImage = "busybox:1.38.0-uclibc"

// These tests need Docker Desktop. If its VM is stopped (e.g. in Resource
// Saver) they wake it rather than skipping.
func requireDocker(t *testing.T) {
	t.Helper()
	s, err := DockerDesktopStatus(context.Background())
	if err != nil {
		t.Skipf("docker desktop not available: %v", err)
	}
	if s.Status != "running" {
		out, err := exec.Command("docker", "desktop", "start", "--timeout", "120").CombinedOutput()
		if err != nil {
			t.Fatalf("waking docker desktop from %q: %v: %s", s.Status, err, out)
		}
	}
}

func TestRunJobOutput(t *testing.T) {
	requireDocker(t)
	res, err := RunJob(context.Background(), 30*time.Second, testImage,
		[]string{"sh", "-c", "echo out; echo err >&2"}, JobOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Stdout) != "out\n" || string(res.Stderr) != "err\n" || res.ExitCode != 0 {
		t.Errorf("got %+v", res)
	}
}

func TestRunJobNonZeroExit(t *testing.T) {
	requireDocker(t)
	res, err := RunJob(context.Background(), 30*time.Second, testImage,
		[]string{"sh", "-c", "echo oops >&2; exit 3"}, JobOptions{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if res.ExitCode != 3 || string(res.Stderr) != "oops\n" {
		t.Errorf("got %+v", res)
	}
}

func TestRunJobTimeout(t *testing.T) {
	requireDocker(t)
	start := time.Now()
	res, err := RunJob(context.Background(), 2*time.Second, testImage,
		[]string{"sh", "-c", "echo started; sleep 60"}, JobOptions{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got err %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("took %v to time out", elapsed)
	}
	if string(res.Stdout) != "started\n" {
		t.Errorf("got stdout %q, want partial output", res.Stdout)
	}
}

func TestRunJobHostOptions(t *testing.T) {
	requireDocker(t)
	res, err := RunJob(context.Background(), 30*time.Second, testImage,
		[]string{"nsenter", "-t", "1", "-m", "--", "cat", "/proc/1/comm"}, JobOptions{Privileged: true, HostPID: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stdout) == 0 {
		t.Error("expected output from the VM's PID 1")
	}
}
