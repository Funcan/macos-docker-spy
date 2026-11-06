# docker-spy

A menu bar app and a CLI showing the Docker Desktop VM's disk, memory, load
and CPU on macOS, to spot when it is getting overloaded.

## Install

```sh
go install github.com/Funcan/macos-docker-spy/cmd/macos-docker-spy@latest
go install github.com/Funcan/macos-docker-spy/cmd/probe@latest
```

The menu bar app uses cgo, so needs the Xcode command line tools
(`xcode-select --install`).

Or download both from the
[releases](https://github.com/Funcan/macos-docker-spy/releases). They aren't
signed or notarised, so macOS will refuse to run them until you remove the
quarantine flag:

```sh
xattr -d com.apple.quarantine macos-docker-spy probe
```

`probe --version` and `macos-docker-spy --version` print the installed
version; the menu bar app also shows it under **About**.

## Menu bar app

```sh
macos-docker-spy
```

The icon shows the VM's state:

- **Whale alone**: Docker Desktop is stopped (including Resource Saver mode),
  or no containers are running.
- **Whale on green**: containers are running; the menu shows live stats.
- **Whale on yellow**: a stat is over its warning threshold, and is marked
  ⚠️ in the menu. While idle, only the last disk usage is checked.
- **Whale on red**: something went wrong; the menu shows the error.

Stats refresh every 10 seconds by default, but only while Docker Desktop is
running and other containers are using it. Collecting them runs a container,
which would otherwise wake a stopped VM, or keep Resource Saver from ever
stopping it. When idle, the menu still shows the last disk usage.

**Preferences…** in the menu has:

- **Start at login**, which adds a LaunchAgent at
  `~/Library/LaunchAgents/com.github.funcan.macos-docker-spy.plist`. It takes
  effect at the next login, and follows the binary if it moves (the app
  updates it each time it starts).
- **Refresh every (seconds)**: at least 5, default 10.
- **Disk, Memory and CPU warning (%)**: the icon turns yellow when a stat is
  above its threshold (defaults 90, 80 and 95). Memory is used / (used +
  buff/cache + free), as the kernel can reclaim buff/cache. CPU is everything
  but idle.

Changes apply straight away. Invalid values are flagged and not saved.

## Probe

```sh
probe
```

```
Disk 25 GB of 32 GB used (84%)  5.0 GB free | 724 MB used 903 MB buff/cache 15 GB free | load 0.28 0.17 0.08 | cpu 0% usr 0% sys 99% idle
```

- Disk is the VM's `/var/lib/docker` filesystem.
- Load is the 1, 5 and 15 minute load averages.
- CPU is sampled over 1 second, across all cores. `usr` includes nice, `sys`
  includes irq/softirq and `idle` includes iowait.
- If the VM is stopped (including Resource Saver mode) it prints
  `VM not running` and exits 1, without waking the VM.

It gathers the stats by running a short-lived `busybox` container that uses
`nsenter` to enter the namespaces of the VM's PID 1, which is why it runs
privileged with the host's PID namespace. The image is pulled on first run.

## Shell inside the VM

```sh
docker run -it --rm --privileged --pid=host busybox:1.38.0-uclibc nsenter -t 1 -m -u -n -i -- sh
```

Replace `sh` with a command to run that instead, e.g. `free -m`.

## Development

- `cmd/` has one directory per binary; `libdocker/` is the shared library
  for querying Docker Desktop and running short-lived containers.
- `make build` builds all binaries into `build/`; `make build-<binary>`
  (e.g. `make build-probe`) builds just one.
- `cmd/macos-docker-spy/whale.svg` is the menu bar icon. Fyne drops
  `fill-rule` when tinting it for the menu bar, so make holes by winding them
  the opposite way to the shape around them.
- `make test`, `make lint`, `make format` and `make clean` do what they say.
  The `libdocker` tests run containers, and wake the Docker Desktop VM if it
  is stopped.
- Pushing a `v*` tag releases via goreleaser.
