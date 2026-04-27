package runner

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aleksclark/spec-torture/internal/schema"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// DockerManager handles container lifecycle for test runtimes.
type DockerManager struct {
	client *client.Client
	logger *slog.Logger
}

// NewDockerManager creates a new DockerManager connected to the local Docker daemon.
func NewDockerManager(logger *slog.Logger) (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to docker: %w", err)
	}

	return &DockerManager{
		client: cli,
		logger: logger,
	}, nil
}

// StartContainer pulls the image and starts a container for the test runtime.
func (dm *DockerManager) StartContainer(ctx context.Context, imageName string, setup *schema.Setup) (string, error) {
	if imageName == "" {
		return "", fmt.Errorf("no image specified")
	}

	dm.logger.Info("pulling image", "image", imageName)
	reader, err := dm.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("pulling image %s: %w", imageName, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)

	env := make([]string, 0)
	if setup != nil {
		for k, v := range setup.Env {
			env = append(env, k+"="+v)
		}
	}

	var cmd []string
	if setup != nil && len(setup.Command) > 0 {
		cmd = setup.Command
	}

	cfg := &container.Config{
		Image: imageName,
		Cmd:   cmd,
		Env:   env,
		Tty:   false,
		OpenStdin: true,
		AttachStdin: true,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostCfg := &container.HostConfig{}

	dm.logger.Info("creating container", "image", imageName)
	resp, err := dm.client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}

	if err := dm.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = dm.removeContainer(ctx, resp.ID)
		return "", fmt.Errorf("starting container: %w", err)
	}

	dm.logger.Info("container started", "id", resp.ID[:12])
	return resp.ID, nil
}

// StopContainer stops and removes a container.
func (dm *DockerManager) StopContainer(ctx context.Context, containerID string) error {
	dm.logger.Info("stopping container", "id", containerID[:12])

	timeout := 10
	stopOpts := container.StopOptions{Timeout: &timeout}
	if err := dm.client.ContainerStop(ctx, containerID, stopOpts); err != nil {
		dm.logger.Warn("error stopping container", "id", containerID[:12], "error", err)
	}

	return dm.removeContainer(ctx, containerID)
}

func (dm *DockerManager) removeContainer(ctx context.Context, containerID string) error {
	return dm.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

// Close releases the Docker client connection.
func (dm *DockerManager) Close() error {
	return dm.client.Close()
}
