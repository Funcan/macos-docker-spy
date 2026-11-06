// Command macos-docker-spy is a menu bar app showing the state of the Docker
// Desktop VM.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

const refreshInterval = 10 * time.Second

func main() {
	systray.Run(onReady, func() {})
}

type menu struct {
	disk, mem, load, cpu, err *systray.MenuItem
	greenIcon, redIcon        []byte
	lastDisk                  string
}

func onReady() {
	greenIcon, err := icon(green)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		systray.Quit()
		return
	}
	redIcon, err := icon(red)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		systray.Quit()
		return
	}
	systray.SetTemplateIcon(whalePNG, whalePNG)
	systray.SetTooltip("Docker Desktop VM")

	m := &menu{greenIcon: greenIcon, redIcon: redIcon, lastDisk: "-"}
	m.disk = systray.AddMenuItem("", "")
	m.mem = systray.AddMenuItem("", "")
	m.load = systray.AddMenuItem("", "")
	m.cpu = systray.AddMenuItem("", "")
	m.err = systray.AddMenuItem("", "")
	for _, item := range []*systray.MenuItem{m.disk, m.mem, m.load, m.cpu, m.err} {
		item.Disable()
	}
	m.err.Hide()
	m.set("-", "-", "-", "-")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit macos-docker-spy")
	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()

	// Refreshing in this one goroutine means refreshes never overlap; ticks
	// that arrive during a slow refresh are dropped.
	go func() {
		m.refresh()
		for range time.Tick(refreshInterval) {
			m.refresh()
		}
	}()
}

// refresh updates the icon and menu. It never wakes the VM: stats are only
// collected when Docker Desktop is running and other containers are using it,
// so our own stats jobs don't keep Resource Saver from stopping it.
func (m *menu) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	status, err := libdocker.DockerDesktopStatus(ctx)
	if errors.Is(err, libdocker.ErrDesktopNotRunning) || (err == nil && status.Status != "running") {
		systray.SetTemplateIcon(whalePNG, whalePNG)
		m.set("-", "-", "-", "-")
		m.err.Hide()
		return
	}
	if err != nil {
		m.fail(err)
		return
	}

	busy, err := libdocker.OtherContainersRunning(ctx)
	if err != nil {
		m.fail(err)
		return
	}
	if !busy {
		systray.SetTemplateIcon(whalePNG, whalePNG)
		m.set(m.lastDisk, "(idle)", "(idle)", "(idle)")
		m.err.Hide()
		return
	}

	stats, err := libdocker.CollectVMStats(ctx)
	if err != nil {
		m.fail(err)
		return
	}
	systray.SetIcon(m.greenIcon)
	disk, mem, load, cpu := statLines(stats)
	m.lastDisk = disk
	m.set(disk, mem, load, cpu)
	m.err.Hide()
}

func (m *menu) set(disk, mem, load, cpu string) {
	m.disk.SetTitle("Disk: " + disk)
	m.mem.SetTitle("Memory: " + mem)
	m.load.SetTitle("Load: " + load)
	m.cpu.SetTitle("CPU: " + cpu)
}

func (m *menu) fail(err error) {
	systray.SetIcon(m.redIcon)
	m.set("-", "-", "-", "-")
	// Job errors can include a container's multi-line stderr.
	first, _, _ := strings.Cut(err.Error(), "\n")
	m.err.SetTitle("Error: " + first)
	m.err.Show()
}

// statLines formats stats for the disk, memory, load and CPU menu items.
func statLines(s libdocker.VMStats) (disk, mem, load, cpu string) {
	disk = fmt.Sprintf("%s of %s used (%d%%)",
		libdocker.HumanKB(s.DiskUsedKB), libdocker.HumanKB(s.DiskTotalKB), s.DiskUsedPercent)
	mem = fmt.Sprintf("%s used, %s buff/cache, %s free",
		libdocker.HumanKB(s.MemUsedKB), libdocker.HumanKB(s.MemBuffCacheKB), libdocker.HumanKB(s.MemFreeKB))
	load = fmt.Sprintf("%.2f %.2f %.2f", s.Load1, s.Load5, s.Load15)
	cpu = fmt.Sprintf("%d%% usr, %d%% sys, %d%% idle", int(s.CPUUser), int(s.CPUSystem), int(s.CPUIdle))
	return disk, mem, load, cpu
}
