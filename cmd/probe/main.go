// Command probe prints a one-line summary of the Docker Desktop VM's disk,
// memory, load and CPU usage.
package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

const image = "busybox:1.38.0-uclibc"

// Runs in the VM's namespaces. The two /proc/stat samples a second apart give
// CPU usage over that second.
const script = `df -k /var/lib/docker | tail -n 1; free -k | grep "^Mem:"; cat /proc/loadavg
grep "^cpu " /proc/stat; sleep 1; grep "^cpu " /proc/stat`

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

	// Enter the namespaces of the VM's PID 1.
	args := []string{"nsenter", "-t", "1", "-m", "-u", "-n", "-i", "--", "sh", "-c", script}
	res, err := libdocker.RunJob(ctx, 10*time.Second, image, args, libdocker.JobOptions{Privileged: true, HostPID: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	summary, err := summarise(string(res.Stdout))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(summary)
}

// summarise turns the output of script into the one-line summary.
func summarise(out string) (string, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		return "", fmt.Errorf("expected 5 lines of output, got %d: %q", len(lines), out)
	}
	df := strings.Fields(lines[0])
	mem := strings.Fields(lines[1])
	load := strings.Fields(lines[2])
	cpu1 := strings.Fields(lines[3])
	cpu2 := strings.Fields(lines[4])
	if len(df) < 5 || len(mem) < 7 || len(load) < 3 || len(cpu1) < 9 || len(cpu2) < 9 {
		return "", fmt.Errorf("unexpected output: %q", out)
	}

	dfKB, err := numbers(df[1:4])
	if err != nil {
		return "", err
	}
	memKB, err := numbers(mem[1:7])
	if err != nil {
		return "", err
	}
	// /proc/stat cpu fields: user nice system idle iowait irq softirq steal ...
	before, err := numbers(cpu1[1:9])
	if err != nil {
		return "", err
	}
	after, err := numbers(cpu2[1:9])
	if err != nil {
		return "", err
	}
	var d [8]float64
	var total float64
	for i := range d {
		d[i] = after[i] - before[i]
		total += d[i]
	}
	usr := 100 * (d[0] + d[1]) / total
	sys := 100 * (d[2] + d[5] + d[6]) / total
	idle := 100 * (d[3] + d[4]) / total

	return fmt.Sprintf("Disk %s of %s used (%s)  %s free | %s used %s buff/cache %s free | load %s %s %s | cpu %d%% usr %d%% sys %d%% idle",
		human(dfKB[1]), human(dfKB[0]), df[4], human(dfKB[2]),
		human(memKB[1]), human(memKB[4]), human(memKB[2]),
		load[0], load[1], load[2],
		int(usr), int(sys), int(idle)), nil
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

// human formats a size in KB, rounding up as df -h does.
func human(kb float64) string {
	if kb < 1024*1024 {
		return fmt.Sprintf("%d MB", int(math.Ceil(kb/1024)))
	}
	gb := kb / 1024 / 1024
	if gb < 10 {
		return fmt.Sprintf("%.1f GB", math.Ceil(gb*10)/10)
	}
	return fmt.Sprintf("%d GB", int(math.Ceil(gb)))
}
