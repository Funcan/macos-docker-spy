# docker-spy

Quick CLI view of the Docker Desktop VM's disk, memory, load and CPU on macOS,
to spot when it is getting overloaded.

## Install

```sh
go install github.com/Funcan/macos-docker-spy/cmd/probe@latest
```

Or download a binary from the
[releases](https://github.com/Funcan/macos-docker-spy/releases).

## Usage

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
- `make build` builds all binaries into `build/`; `make build-probe` builds
  just the probe.
- `make test`, `make lint`, `make format` and `make clean` do what they say.
  The `libdocker` tests run containers, and wake the Docker Desktop VM if it
  is stopped.
- Pushing a `v*` tag releases via goreleaser.
