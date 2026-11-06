package libdocker

import (
	"errors"
	"os/exec"
	"testing"
)

func TestParseDesktopStatus(t *testing.T) {
	out := []byte(`{
    "SessionID": "835f2462-7694-4f28-98ce-895b0bfdef97",
    "Status": "running"
}
`)
	got, err := parseDesktopStatus(out)
	if err != nil {
		t.Fatal(err)
	}
	want := DesktopStatus{Status: "running", SessionID: "835f2462-7694-4f28-98ce-895b0bfdef97"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseDesktopStatusInvalid(t *testing.T) {
	if _, err := parseDesktopStatus([]byte("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestStatusErrorNotRunning(t *testing.T) {
	exitErr := exec.Command("sh", "-c", "exit 1").Run()
	stderr := "Could not retrieve status. Is Docker Desktop running?You can start Docker Desktop by running 'docker desktop start'.\n"
	if err := statusError(exitErr, stderr); !errors.Is(err, ErrDesktopNotRunning) {
		t.Errorf("got %v, want ErrDesktopNotRunning", err)
	}
	if err := statusError(exitErr, "something else broke"); errors.Is(err, ErrDesktopNotRunning) {
		t.Errorf("got ErrDesktopNotRunning for an unrelated failure")
	}
}
