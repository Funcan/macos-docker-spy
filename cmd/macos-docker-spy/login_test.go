package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetStartAtLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "LaunchAgents", agentLabel+".plist")
	exe := "/Applications/A & B/macos-docker-spy"

	if err := setStartAtLogin(path, exe, true); err != nil {
		t.Fatal(err)
	}
	if !startsAtLogin(path) {
		t.Fatal("not enabled after turning on")
	}
	if out, err := exec.Command("plutil", "-lint", path).CombinedOutput(); err != nil {
		t.Fatalf("invalid plist: %v: %s", err, out)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<string>/Applications/A &amp; B/macos-docker-spy</string>") {
		t.Errorf("executable path missing or unescaped:\n%s", b)
	}

	for range 2 { // turning off twice is fine
		if err := setStartAtLogin(path, exe, false); err != nil {
			t.Fatal(err)
		}
	}
	if startsAtLogin(path) {
		t.Error("still enabled after turning off")
	}
}
