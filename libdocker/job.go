package libdocker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// JobOptions are extra settings for the container RunJob starts.
type JobOptions struct {
	Privileged bool // docker run --privileged
	HostPID    bool // docker run --pid=host
}

// JobResult is what a job printed and how it exited.
type JobResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// RunJob runs image with args in a new container, waits for it to exit and
// returns its output. The image is pulled first if it isn't present. If the
// container doesn't exit within timeout (which doesn't include the pull) it is
// killed and the output so far is returned with an error wrapping
// context.DeadlineExceeded. A non-zero exit code also returns an error,
// alongside the populated result. The container is always removed.
func RunJob(ctx context.Context, timeout time.Duration, image string, args []string, opts JobOptions) (JobResult, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return JobResult{}, fmt.Errorf("docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	if _, err := cli.ImageInspect(ctx, image); err != nil {
		pull, err := cli.ImagePull(ctx, image, client.ImagePullOptions{})
		if err != nil {
			return JobResult{}, fmt.Errorf("pulling %s: %w", image, err)
		}
		if err := pull.Wait(ctx); err != nil {
			return JobResult{}, fmt.Errorf("pulling %s: %w", image, err)
		}
	}

	hostConfig := &container.HostConfig{Privileged: opts.Privileged}
	if opts.HostPID {
		hostConfig.PidMode = "host"
	}
	created, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     &container.Config{Image: image, Cmd: args, AttachStdout: true, AttachStderr: true},
		HostConfig: hostConfig,
	})
	if err != nil {
		return JobResult{}, fmt.Errorf("creating container: %w", err)
	}
	// Not AutoRemove, which can race with collecting the output. Use a fresh
	// context so cleanup still happens after a timeout or cancellation.
	defer func() {
		rmCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_, _ = cli.ContainerRemove(rmCtx, created.ID, client.ContainerRemoveOptions{Force: true})
	}()

	attach, err := cli.ContainerAttach(ctx, created.ID, client.ContainerAttachOptions{Stream: true, Stdout: true, Stderr: true})
	if err != nil {
		return JobResult{}, fmt.Errorf("attaching to container: %w", err)
	}
	defer attach.Close()
	var stdout, stderr bytes.Buffer
	copied := make(chan struct{})
	go func() {
		_, _ = stdcopy.StdCopy(&stdout, &stderr, attach.Reader)
		close(copied)
	}()

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	wait := cli.ContainerWait(runCtx, created.ID, client.ContainerWaitOptions{Condition: container.WaitConditionNextExit})
	if _, err := cli.ContainerStart(runCtx, created.ID, client.ContainerStartOptions{}); err != nil {
		return JobResult{}, fmt.Errorf("starting container: %w", err)
	}

	select {
	case exit := <-wait.Result:
		<-copied
		result := JobResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: int(exit.StatusCode)}
		if exit.StatusCode != 0 {
			return result, fmt.Errorf("container exited with code %d: %s", exit.StatusCode, strings.TrimSpace(stderr.String()))
		}
		return result, nil
	case err := <-wait.Error:
		if runCtx.Err() == nil {
			return JobResult{}, fmt.Errorf("waiting for container: %w", err)
		}
		// The wait was cut short by the timeout; fall through to kill it.
	case <-runCtx.Done():
	}

	killCtx, killCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer killCancel()
	_, _ = cli.ContainerKill(killCtx, created.ID, client.ContainerKillOptions{})
	attach.Close()
	<-copied
	result := JobResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: -1}
	return result, fmt.Errorf("container did not finish: %w", runCtx.Err())
}
