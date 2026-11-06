// Command macos-docker-spy is a menu bar app showing the state of the Docker
// Desktop VM.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
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

// Preference keys and defaults.
const (
	prefRefreshSeconds    = "refreshSeconds"
	defaultRefreshSeconds = 10
	minRefreshSeconds     = 5

	prefDiskWarnPercent    = "diskWarnPercent"
	defaultDiskWarnPercent = 90
	prefMemWarnPercent     = "memWarnPercent"
	defaultMemWarnPercent  = 80
	prefCPUWarnPercent     = "cpuWarnPercent"
	defaultCPUWarnPercent  = 95
)

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
	yellowIcon, err := icon(yellow)
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
		tray:       tray,
		settings:   a.Preferences(),
		greenIcon:  fyne.NewStaticResource("green.png", greenIcon),
		yellowIcon: fyne.NewStaticResource("yellow.png", yellowIcon),
		redIcon:    fyne.NewStaticResource("red.png", redIcon),
		menu:       fyne.NewMenu("Docker Desktop VM"),
		disk:       &fyne.MenuItem{Label: "Disk: -", Disabled: true},
		mem:        &fyne.MenuItem{Label: "Memory: -", Disabled: true},
		load:       &fyne.MenuItem{Label: "Load: -", Disabled: true},
		cpu:        &fyne.MenuItem{Label: "CPU: -", Disabled: true},
		err:        &fyne.MenuItem{Disabled: true},
		// Fyne only adds its own Quit after drawing the menu, so it would be
		// missing; with IsQuit set, Fyne makes this one quit the app.
		quit: &fyne.MenuItem{Label: "Quit", IsQuit: true},
		prefs: fyne.NewMenuItem("Preferences…", func() {
			prefs.Show()
			prefs.RequestFocus()
		}),
		lastDisk:        "-",
		lastDiskPercent: -1,
	}
	m.menu.Items = []*fyne.MenuItem{m.disk, m.mem, m.load, m.cpu, fyne.NewMenuItemSeparator(), m.prefs, m.quit}
	tray.SetSystemTrayMenu(m.menu)
	tray.SetSystemTrayIcon(whaleTemplate)

	// Changing a preference refreshes straight away, so the new period (and
	// later settings) take effect without waiting out the old period.
	changed := make(chan struct{}, 1)
	a.Preferences().AddChangeListener(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})

	// Refreshing in this one goroutine means refreshes never overlap.
	a.Lifecycle().SetOnStarted(func() {
		hideFromDock()
		go func() {
			for {
				m.refresh()
				period := a.Preferences().IntWithFallback(prefRefreshSeconds, defaultRefreshSeconds)
				select {
				case <-time.After(time.Duration(period) * time.Second):
				case <-changed:
				}
			}
		}()
	})
	a.Run()
}

type menu struct {
	tray                           desktop.App
	settings                       fyne.Preferences
	greenIcon, yellowIcon, redIcon fyne.Resource
	menu                           *fyne.Menu
	disk, mem, load, cpu, err      *fyne.MenuItem
	prefs, quit                    *fyne.MenuItem
	lastDisk                       string
	// -1 until the disk has been measured.
	lastDiskPercent int
}

// warnings says which stats are over their warning thresholds.
type warnings struct{ disk, mem, cpu bool }

// refresh updates the icon and menu. It never wakes the VM: stats are only
// collected when Docker Desktop is running and other containers are using it,
// so our own stats jobs don't keep Resource Saver from stopping it.
func (m *menu) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	status, err := libdocker.DockerDesktopStatus(ctx)
	if errors.Is(err, libdocker.ErrDesktopNotRunning) || (err == nil && status.Status != "running") {
		m.show(whaleTemplate, "-", "-", "-", "-", warnings{}, nil)
		return
	}
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", warnings{}, err)
		return
	}

	diskWarn := m.settings.IntWithFallback(prefDiskWarnPercent, defaultDiskWarnPercent)
	memWarn := m.settings.IntWithFallback(prefMemWarnPercent, defaultMemWarnPercent)
	cpuWarn := m.settings.IntWithFallback(prefCPUWarnPercent, defaultCPUWarnPercent)

	busy, err := libdocker.OtherContainersRunning(ctx)
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", warnings{}, err)
		return
	}
	if !busy {
		// The disk keeps filling while idle, so its last value still counts.
		w := warnings{disk: m.lastDiskPercent > diskWarn}
		var icon fyne.Resource = whaleTemplate
		if w.disk {
			icon = m.yellowIcon
		}
		m.show(icon, m.lastDisk, "(idle)", "(idle)", "(idle)", w, nil)
		return
	}

	stats, err := libdocker.CollectVMStats(ctx)
	if err != nil {
		m.show(m.redIcon, "-", "-", "-", "-", warnings{}, err)
		return
	}
	disk, mem, load, cpu := statLines(stats)
	m.lastDisk = disk
	m.lastDiskPercent = stats.DiskUsedPercent
	w := overThresholds(stats, diskWarn, memWarn, cpuWarn)
	icon := m.greenIcon
	if w.disk || w.mem || w.cpu {
		icon = m.yellowIcon
	}
	m.show(icon, disk, mem, load, cpu, w, nil)
}

// show sets the icon and menu, marking stats that are over their thresholds
// and adding an error item if err isn't nil. It is safe to call from any
// goroutine.
func (m *menu) show(icon fyne.Resource, disk, mem, load, cpu string, w warnings, err error) {
	fyne.Do(func() {
		m.tray.SetSystemTrayIcon(icon)
		m.disk.Label = warnPrefix(w.disk) + "Disk: " + disk
		m.mem.Label = warnPrefix(w.mem) + "Memory: " + mem
		m.load.Label = "Load: " + load
		m.cpu.Label = warnPrefix(w.cpu) + "CPU: " + cpu
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

	refresh := widget.NewEntry()
	refresh.SetText(strconv.Itoa(a.Preferences().IntWithFallback(prefRefreshSeconds, defaultRefreshSeconds)))
	refresh.Validator = func(s string) error {
		_, err := parseRefreshSeconds(s)
		return err
	}
	// Only valid values are saved, so the last good one stays in use.
	refresh.OnChanged = func(s string) {
		if n, err := parseRefreshSeconds(s); err == nil {
			a.Preferences().SetInt(prefRefreshSeconds, n)
		}
	}

	form := widget.NewForm(widget.NewFormItem("Refresh every (seconds)", refresh))
	for _, t := range []struct {
		label, key string
		def        int
	}{
		{"Disk warning (%)", prefDiskWarnPercent, defaultDiskWarnPercent},
		{"Memory warning (%)", prefMemWarnPercent, defaultMemWarnPercent},
		{"CPU warning (%)", prefCPUWarnPercent, defaultCPUWarnPercent},
	} {
		e := widget.NewEntry()
		e.SetText(strconv.Itoa(a.Preferences().IntWithFallback(t.key, t.def)))
		e.Validator = func(s string) error {
			_, err := parsePercent(s)
			return err
		}
		e.OnChanged = func(s string) {
			if n, err := parsePercent(s); err == nil {
				a.Preferences().SetInt(t.key, n)
			}
		}
		form.Append(t.label, e)
	}

	// Sized for the settings still to come, rather than shrunk to fit them.
	w.SetContent(container.NewPadded(container.NewVBox(login, form)))
	w.Resize(fyne.NewSize(400, 240))
	w.SetCloseIntercept(w.Hide)
	return w
}

// parseRefreshSeconds parses a refresh period typed into Preferences.
func parseRefreshSeconds(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, errors.New("not a whole number")
	}
	if n < minRefreshSeconds {
		return 0, fmt.Errorf("must be at least %d", minRefreshSeconds)
	}
	return n, nil
}

// parsePercent parses a warning threshold typed into Preferences.
func parsePercent(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, errors.New("not a whole number")
	}
	if n < 0 || n > 100 {
		return 0, errors.New("must be between 0 and 100")
	}
	return n, nil
}

// overThresholds says which of s's stats are strictly above their warning
// thresholds. Memory counts buff/cache as available, as the kernel can
// reclaim it, and CPU is busy for whatever isn't idle.
func overThresholds(s libdocker.VMStats, diskWarn, memWarn, cpuWarn int) warnings {
	var memPercent float64
	if total := s.MemUsedKB + s.MemBuffCacheKB + s.MemFreeKB; total > 0 {
		memPercent = 100 * float64(s.MemUsedKB) / float64(total)
	}
	return warnings{
		disk: s.DiskUsedPercent > diskWarn,
		mem:  memPercent > float64(memWarn),
		cpu:  100-s.CPUIdle > float64(cpuWarn),
	}
}

func warnPrefix(over bool) string {
	if over {
		return "⚠️ "
	}
	return ""
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
