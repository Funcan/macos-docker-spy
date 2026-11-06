package version

import (
	"runtime/debug"
	"testing"
)

func TestFormat(t *testing.T) {
	for _, tt := range []struct {
		name     string
		version  string
		settings map[string]string
		want     string
	}{
		{
			"release", "v0.0.1",
			map[string]string{"vcs.revision": "08ab39f94966c85e7aa8c9baa2ddb01c131ed646", "vcs.time": "2026-11-06T16:45:06Z", "vcs.modified": "false"},
			"probe v0.0.1 (08ab39f, 2026-11-06T16:45:06Z)",
		},
		{
			"dirty pseudo-version", "v0.0.1-0.20261106164506-08ab39f94966+dirty",
			map[string]string{"vcs.revision": "08ab39f94966c85e7aa8c9baa2ddb01c131ed646", "vcs.time": "2026-11-06T16:45:06Z", "vcs.modified": "true"},
			"probe v0.0.1-0.20261106164506-08ab39f94966+dirty (08ab39f, 2026-11-06T16:45:06Z)",
		},
		{
			"dirty without marker", "(devel)",
			map[string]string{"vcs.revision": "08ab39f94966c85e7aa8c9baa2ddb01c131ed646", "vcs.time": "2026-11-06T16:45:06Z", "vcs.modified": "true"},
			"probe (devel) (08ab39f, 2026-11-06T16:45:06Z) (dirty)",
		},
		{
			"no VCS info", "(devel)", nil,
			"probe (devel)",
		},
	} {
		info := &debug.BuildInfo{Main: debug.Module{Version: tt.version}}
		for k, v := range tt.settings {
			info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
		}
		if got := format("probe", info); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
