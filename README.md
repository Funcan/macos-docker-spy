# docker-spy

Quick CLI view of the Docker Desktop VM's disk, memory, load and CPU on macOS,
to spot when it is getting overloaded.

## Usage

```sh
bin/probe
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

The `docker-spy` image is built automatically on first run.

## Shell inside the VM

```sh
docker run -it --rm --privileged --pid=host docker-spy
```

The image is `busybox` with `nsenter` into the namespaces of the VM's PID 1,
which is why it needs `--privileged --pid=host`. Pass a command to run it
instead of a shell, e.g. `docker run --rm --privileged --pid=host docker-spy free -m`.
