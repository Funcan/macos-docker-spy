// Package version describes the running binary's version from the build info
// the Go toolchain embeds, so releases need no -ldflags.
package version

import (
	"runtime/debug"
	"strings"
)

// String returns e.g. "probe v0.0.1 (08ab39f, 2026-11-06T16:45:06Z)".
func String(name string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return name + " (unknown version)"
	}
	return format(name, info)
}

func format(name string, info *debug.BuildInfo) string {
	var rev, at string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			at = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}

	out := name + " " + info.Main.Version
	if rev != "" && at != "" {
		out += " (" + rev + ", " + at + ")"
	} else if rev != "" {
		out += " (" + rev + ")"
	}
	// Versions derived from a dirty checkout already end in "+dirty".
	if dirty && !strings.Contains(info.Main.Version, "dirty") {
		out += " (dirty)"
	}
	return out
}
