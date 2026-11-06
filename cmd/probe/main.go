// Command probe prints a one-line summary of the Docker Desktop VM's disk,
// memory, load and CPU usage.
package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

func main() {
	ctx := context.Background()

	// Check first, as running a container would wake the VM from Resource Saver.
	status, err := libdocker.DockerDesktopStatus(ctx)
	if errors.Is(err, libdocker.ErrDesktopNotRunning) || (err == nil && status.Status != "running") {
		fmt.Println("VM not running")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	stats, err := libdocker.CollectVMStats(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(summarise(stats))
}

func summarise(s libdocker.VMStats) string {
	return fmt.Sprintf("Disk %s of %s used (%d%%)  %s free | %s used %s buff/cache %s free | load %.2f %.2f %.2f | cpu %d%% usr %d%% sys %d%% idle",
		human(s.DiskUsedKB), human(s.DiskTotalKB), s.DiskUsedPercent, human(s.DiskFreeKB),
		human(s.MemUsedKB), human(s.MemBuffCacheKB), human(s.MemFreeKB),
		s.Load1, s.Load5, s.Load15,
		int(s.CPUUser), int(s.CPUSystem), int(s.CPUIdle))
}

// human formats a size in KB, rounding up as df -h does.
func human(kb int64) string {
	if kb < 1024*1024 {
		return fmt.Sprintf("%d MB", int(math.Ceil(float64(kb)/1024)))
	}
	gb := float64(kb) / 1024 / 1024
	if gb < 10 {
		return fmt.Sprintf("%.1f GB", math.Ceil(gb*10)/10)
	}
	return fmt.Sprintf("%d GB", int(math.Ceil(gb)))
}
