package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const agentLabel = "com.github.funcan.macos-docker-spy"

// agentPath is where the LaunchAgent that starts the app at login lives.
func agentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist"), nil
}

// executable is the path of the running binary, with symlinks resolved so the
// LaunchAgent still works if a link is removed.
func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func startsAtLogin(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// setStartAtLogin writes or removes the LaunchAgent at path. It doesn't load
// or unload it with launchctl, which would start a second copy or stop this
// one; the change takes effect at the next login.
func setStartAtLogin(path, exe string, on bool) error {
	if !on {
		err := os.Remove(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(exe)); err != nil {
		return err
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + agentLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + escaped.String() + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
	return os.WriteFile(path, []byte(plist), 0o644)
}
