package main

import (
	"testing"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

// Golden output is from the original shell/awk version of the probe, given
// the same numbers.
func TestSummarise(t *testing.T) {
	tests := []struct {
		name  string
		stats libdocker.VMStats
		want  string
	}{
		{
			name: "real VM numbers",
			stats: libdocker.VMStats{
				DiskTotalKB: 32846608, DiskUsedKB: 27429352, DiskFreeKB: 3723204, DiskUsedPercent: 89,
				MemUsedKB: 638168, MemBuffCacheKB: 270780, MemFreeKB: 15646468,
				Load1: 0.36, Load5: 0.08, Load15: 0.03,
				CPUUser: 0, CPUSystem: 100 * 1.0 / 804, CPUIdle: 100 * 803.0 / 804,
			},
			want: "Disk 27 GB of 32 GB used (89%)  3.6 GB free | 624 MB used 265 MB buff/cache 15 GB free | load 0.36 0.08 0.03 | cpu 0% usr 0% sys 99% idle",
		},
		{
			name: "busy CPU and small sizes",
			stats: libdocker.VMStats{
				DiskTotalKB: 900000, DiskUsedKB: 500, DiskFreeKB: 399500, DiskUsedPercent: 1,
				MemUsedKB: 1100, MemBuffCacheKB: 100, MemFreeKB: 800,
				Load1: 1, Load5: 2, Load15: 3,
				CPUUser: 100 * 220.0 / 709, CPUSystem: 100 * 65.0 / 709, CPUIdle: 100 * 420.0 / 709,
			},
			want: "Disk 1 MB of 879 MB used (1%)  391 MB free | 2 MB used 1 MB buff/cache 1 MB free | load 1.00 2.00 3.00 | cpu 31% usr 9% sys 59% idle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarise(tt.stats); got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}
