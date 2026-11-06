package libdocker

import (
	"context"
	"testing"
)

func TestParseVMStats(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want VMStats
	}{
		{
			name: "real VM output",
			out: `/dev/vda1       32846608 27429352   3723204  89% /var/lib/docker
Mem:        16356724      638168    15646468         508      270780    15718556
0.36 0.08 0.03 3/253 1489
cpu  194 0 118 67514 57 0 104 0 0 0
cpu  194 0 119 68317 57 0 104 0 0 0
`,
			want: VMStats{
				DiskTotalKB: 32846608, DiskUsedKB: 27429352, DiskFreeKB: 3723204, DiskUsedPercent: 89,
				MemUsedKB: 638168, MemBuffCacheKB: 270780, MemFreeKB: 15646468,
				Load1: 0.36, Load5: 0.08, Load15: 0.03,
				CPUUser: 0, CPUSystem: 100 * 1.0 / 804, CPUIdle: 100 * 803.0 / 804,
			},
		},
		{
			// Every /proc/stat field moves, to check how they're grouped.
			name: "busy CPU",
			out: `/dev/vda1 900000 500 399500 1% /var/lib/docker
Mem: 2000 1100 800 0 100 900
1.00 2.00 3.00 1/1 1
cpu  100 10 50 800 40 5 5 3 0 0
cpu  300 30 100 1200 60 10 15 7 0 0
`,
			want: VMStats{
				DiskTotalKB: 900000, DiskUsedKB: 500, DiskFreeKB: 399500, DiskUsedPercent: 1,
				MemUsedKB: 1100, MemBuffCacheKB: 100, MemFreeKB: 800,
				Load1: 1, Load5: 2, Load15: 3,
				CPUUser: 100 * 220.0 / 709, CPUSystem: 100 * 65.0 / 709, CPUIdle: 100 * 420.0 / 709,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVMStats(tt.out)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestParseVMStatsBadOutput(t *testing.T) {
	if _, err := parseVMStats("nsenter: failed\n"); err == nil {
		t.Error("expected an error")
	}
}

func TestCollectVMStats(t *testing.T) {
	requireDocker(t)
	s, err := CollectVMStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.DiskTotalKB == 0 || s.MemFreeKB == 0 || s.CPUIdle == 0 {
		t.Errorf("implausible stats: %+v", s)
	}
}
