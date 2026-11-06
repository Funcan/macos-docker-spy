# docker-spy

Quick CLI view of the Docker Desktop VM's disk, memory and load on macOS, to
spot when it is getting overloaded.

## Usage

```sh
bin/probe
```

```
Disk 25 GB of 32 GB used (83%)  5.3 GB free | 482 MB used 264 MB buff/cache 7.1 GB free | load 0.00 0.00 0.00
```

- Disk is the VM's `/var/lib/docker` filesystem.
- Load is the 1, 5 and 15 minute load averages.
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
