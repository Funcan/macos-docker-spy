package libdocker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const vmStatsImage = "busybox:1.38.0-uclibc"

// Runs in the VM's namespaces. The two /proc/stat samples a second apart give
// CPU usage over that second.
const vmStatsScript = `df -k /var/lib/docker | tail -n 1; free -k | grep "^Mem:"; cat /proc/loadavg
grep "^cpu " /proc/stat; sleep 1; grep "^cpu " /proc/stat`

// VMStats is a snapshot of the Docker Desktop VM's resource usage. Sizes are
// in KB and CPU figures are percentages of all cores.
type VMStats struct {
	// The /var/lib/docker filesystem, as reported by df.
	DiskTotalKB, DiskUsedKB, DiskFreeKB int64
	DiskUsedPercent                     int

	MemUsedKB, MemBuffCacheKB, MemFreeKB int64

	Load1, Load5, Load15 float64

	// User includes nice, System includes irq/softirq and Idle includes
	// iowait. Steal is left out, so they may not add up to 100.
	CPUUser, CPUSystem, CPUIdle float64
}

// CollectVMStats gathers VMStats by running a short-lived privileged busybox
// container in the VM's namespaces. It takes just over a second, to sample CPU
// usage. It wakes the VM if it is stopped, so callers that want to avoid that
// should check DockerDesktopStatus first.
func CollectVMStats(ctx context.Context) (VMStats, error) {
	// Enter the namespaces of the VM's PID 1.
	args := []string{"nsenter", "-t", "1", "-m", "-u", "-n", "-i", "--", "sh", "-c", vmStatsScript}
	res, err := RunJob(ctx, 10*time.Second, vmStatsImage, args, JobOptions{Privileged: true, HostPID: true})
	if err != nil {
		return VMStats{}, err
	}
	return parseVMStats(string(res.Stdout))
}

func parseVMStats(out string) (VMStats, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		return VMStats{}, fmt.Errorf("expected 5 lines of output, got %d: %q", len(lines), out)
	}
	df := strings.Fields(lines[0])
	mem := strings.Fields(lines[1])
	load := strings.Fields(lines[2])
	cpu1 := strings.Fields(lines[3])
	cpu2 := strings.Fields(lines[4])
	if len(df) < 5 || len(mem) < 7 || len(load) < 3 || len(cpu1) < 9 || len(cpu2) < 9 {
		return VMStats{}, fmt.Errorf("unexpected output: %q", out)
	}

	dfKB, err := numbers(df[1:4])
	if err != nil {
		return VMStats{}, err
	}
	diskPercent, err := strconv.Atoi(strings.TrimSuffix(df[4], "%"))
	if err != nil {
		return VMStats{}, fmt.Errorf("parsing %q: %w", df[4], err)
	}
	memKB, err := numbers(mem[1:7])
	if err != nil {
		return VMStats{}, err
	}
	loads, err := numbers(load[0:3])
	if err != nil {
		return VMStats{}, err
	}
	// /proc/stat cpu fields: user nice system idle iowait irq softirq steal ...
	before, err := numbers(cpu1[1:9])
	if err != nil {
		return VMStats{}, err
	}
	after, err := numbers(cpu2[1:9])
	if err != nil {
		return VMStats{}, err
	}
	var d [8]float64
	var total float64
	for i := range d {
		d[i] = after[i] - before[i]
		total += d[i]
	}
	if total == 0 {
		return VMStats{}, fmt.Errorf("no CPU time passed between samples: %q", out)
	}

	return VMStats{
		DiskTotalKB:     int64(dfKB[0]),
		DiskUsedKB:      int64(dfKB[1]),
		DiskFreeKB:      int64(dfKB[2]),
		DiskUsedPercent: diskPercent,
		MemUsedKB:       int64(memKB[1]),
		MemBuffCacheKB:  int64(memKB[4]),
		MemFreeKB:       int64(memKB[2]),
		Load1:           loads[0],
		Load5:           loads[1],
		Load15:          loads[2],
		CPUUser:         100 * (d[0] + d[1]) / total,
		CPUSystem:       100 * (d[2] + d[5] + d[6]) / total,
		CPUIdle:         100 * (d[3] + d[4]) / total,
	}, nil
}

func numbers(fields []string) ([]float64, error) {
	var n []float64
	for _, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", f, err)
		}
		n = append(n, v)
	}
	return n, nil
}
