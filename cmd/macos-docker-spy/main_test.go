package main

import (
	"testing"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

func TestStatLines(t *testing.T) {
	disk, mem, load, cpu := statLines(libdocker.VMStats{
		DiskTotalKB: 32846608, DiskUsedKB: 27429352, DiskFreeKB: 3723204, DiskUsedPercent: 89,
		MemUsedKB: 638168, MemBuffCacheKB: 270780, MemFreeKB: 15646468,
		Load1: 0.36, Load5: 0.08, Load15: 0.03,
		CPUUser: 100 * 220.0 / 709, CPUSystem: 100 * 65.0 / 709, CPUIdle: 100 * 420.0 / 709,
	})
	for _, tt := range []struct{ got, want string }{
		{disk, "27 GB of 32 GB used (89%)"},
		{mem, "624 MB used, 265 MB buff/cache, 15 GB free"},
		{load, "0.36 0.08 0.03"},
		{cpu, "31% usr, 9% sys, 59% idle"},
	} {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestParseRefreshSeconds(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int
		ok   bool
	}{
		{"10", 10, true},
		{" 5 ", 5, true},
		{"4", 0, false},
		{"", 0, false},
		{"2.5", 0, false},
		{"ten", 0, false},
	} {
		got, err := parseRefreshSeconds(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("parseRefreshSeconds(%q) = %d, %v", tt.in, got, err)
		}
	}
}
