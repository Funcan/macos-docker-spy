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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/Funcan/macos-docker-spy/libdocker"
)

const refreshInterval = 10 * time.Second

func main() {
	a := app.NewWithID("com.github.funcan.macos-docker-spy")
	tray, ok := a.(desktop.App)
	if !ok {
		fmt.Fprintln(os.Stderr, "no menu bar support")
		os.Exit(1)
	}
	greenIcon, err := icon(green)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	redIcon, err := icon(red)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent, err := agentPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	exe, err := executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Keep the LaunchAgent pointing at this binary, in case it has moved.
	if startsAtLogin(agent) {
		if err := setStartAtLogin(agent, exe, true); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	prefs := preferencesWindow(a, agent, exe)

	m := &menu{
		tray:      tray,
		greenIcon: fyne.NewStaticResource("green.png", greenIcon),
		redIcon:   fyne.NewStaticResource("red.png", redIcon),
		menu:      fyne.NewMenu("Docker Desktop VM"),
		disk:      &fyne.MenuItem{Label: "Disk: -", Disabled: true},
		mem:       &fyne.MenuItem{Label: "Memory: -", Disabled: true},
		load:      &fyne.MenuItem{Label: "Load: -", Disabled: true},
		cpu:       &fyne.MenuItem{Label: "CPU: -", Disabled: true},
		err:       &fyne.MenuItem{Disabled: true},
		// Fyne only adds its own Quit after drawing the menu, so it would be
		// missing; with IsQuit set, Fyne makes this one quit the app.
		quit: &fyne.MenuItem{Label: "Quit", IsQuit: true},
		prefs: fyne.NewMenuItem("Preferences…", func() {
			prefs.Show()
			prefs.RequestFocus()
		}),
		lastDisk: "-",
	}
	m.menu.Items = []*fyne.MenuItem{m.disk, m.mem, m.load, m.cpu, fyne.NewMenuItemSeparator(), m.prefs, m.quit}
	tray.SetSystemTrayMenu(m.menu)
	tray.SetSystemTrayIcon(whaleTemplate)

	// Refreshing in this one goroutine means refreshes never overlap; ticks
	// that arrive during a slow refresh are dropped.
	a.Lifecycle().SetOnStarted(func() {
		hideFromDock()
		go func() {
			m.refresh()
			for range time.Tick(refreshInterval) {
				m.refresh()
			}
		}()
	})
	a.Run()
}

type menu struct {
	tray                      desktop.App
	greenIcon, redIcon        fyne.Resource
	menu                      *fyne.Menu
	disk, mem, load, cpu, err *fyne.MenuItem
	prefs, quit               *fyne.MenuItem
	lastDisk                  string
}

// refresh updates the icon and menu. It never wakes the VM: stats are only
// collected when Docker Desktop is running and other containers are using it,
// so our own stats jobs don't keep Resource Saver from stopping it.
func (m *menu) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	status, err := libdocker.DockerDesktopStatus(ctx)
	if errors.Is(err, libdocker.ErrDesktopNotRunning) || (err == nil && status.Status != "running") {
		m.show(whaleTemplate, "-", "-", "-", "-", nil)
		return
	}
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", err)
		return
	}

	busy, err := libdocker.OtherContainersRunning(ctx)
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", err)
		return
	}
	if !busy {
		m.show(whaleTemplate, m.lastDisk, "(idle)", "(idle)", "(idle)", nil)
		return
	}

	stats, err := libdocker.CollectVMStats(ctx)
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", err)
		return
	}
	disk, mem, load, cpu := statLines(stats)
	m.lastDisk = disk
	m.show(m.greenIcon, disk, mem, load, cpu, nil)
}

// show sets the icon and menu, adding an error item if err isn't nil. It is
// safe to call from any goroutine.
func (m *menu) show(icon fyne.Resource, disk, mem, load, cpu string, err error) {
	fyne.Do(func() {
		m.tray.SetSystemTrayIcon(icon)
		m.disk.Label = "Disk: " + disk
		m.mem.Label = "Memory: " + mem
		m.load.Label = "Load: " + load
		m.cpu.Label = "CPU: " + cpu
		m.menu.Items = []*fyne.MenuItem{m.disk, m.mem, m.load, m.cpu}
		if err != nil {
			// Job errors can include a container's multi-line stderr.
			first, _, _ := strings.Cut(err.Error(), "\n")
			m.err.Label = "Error: " + first
			m.menu.Items = append(m.menu.Items, m.err)
		}
		m.menu.Items = append(m.menu.Items, fyne.NewMenuItemSeparator(), m.prefs, m.quit)
		m.menu.Refresh()
	})
}

// preferencesWindow returns a hidden window for changing settings. Closing it
// hides it again.
func preferencesWindow(a fyne.App, agent, exe string) fyne.Window {
	w := a.NewWindow("macos-docker-spy Preferences")
	login := widget.NewCheck("Start at login", nil)
	login.Checked = startsAtLogin(agent)
	login.OnChanged = func(on bool) {
		if err := setStartAtLogin(agent, exe, on); err != nil {
			dialog.ShowError(err, w)
			login.Checked = !on
			login.Refresh()
		}
	}
	// Sized for the settings still to come, rather than shrunk to fit one.
	w.SetContent(container.NewPadded(container.NewVBox(login)))
	w.Resize(fyne.NewSize(400, 240))
	w.SetCloseIntercept(w.Hide)
	return w
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
