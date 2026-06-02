package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type ContainerConfig struct {
	Image      string
	VolumeBind string
	Workspace  string
	EntryPoint []string
}

type ContainerResult struct {
	ContainerID string
	Port        int
}

func allocatePort() int {
	for port := 10001; port < 11000; port++ {
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		ln.Close()
		return port
	}
	return 0
}

func PullImage(ctx context.Context, cli *client.Client, imageRef string) error {
	reader, err := cli.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("image pull %s: %w", imageRef, err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)
	log.Printf("[DOCKER] Pulled image %s", imageRef)
	return nil
}

func LaunchContainer(ctx context.Context, cli *client.Client, cfg ContainerConfig) (*ContainerResult, error) {
	port := allocatePort()
	if port == 0 {
		return nil, fmt.Errorf("no available port found in range 10001-10999")
	}

	hostPort := strconv.Itoa(port)

	containerCfg := &container.Config{
		Image: cfg.Image,
		Cmd:   cfg.EntryPoint,
		Tty:   false,
		OpenStdin: false,
		StdinOnce: false,
		ExposedPorts: nat.PortSet{
			"22/tcp": struct{}{},
		},
	}

	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"22/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: cfg.VolumeBind,
				Target: "/workspace",
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("container start: %w", err)
	}

	hostname, _ := os.Hostname()
	log.Printf("[DOCKER] Launched container %s (image: %s, ssh: %s:%d)",
		resp.ID[:12], cfg.Image, hostname, port)

	return &ContainerResult{
		ContainerID: resp.ID,
		Port:        port,
	}, nil
}

func StopContainer(ctx context.Context, cli *client.Client, containerID string) error {
	timeoutSec := 10
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeoutSec}); err != nil {
		return fmt.Errorf("container stop %s: %w", containerID, err)
	}
	log.Printf("[DOCKER] Stopped container %s", containerID[:12])
	return nil
}

func RemoveContainer(ctx context.Context, cli *client.Client, containerID string) error {
	if err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("container remove %s: %w", containerID, err)
	}
	log.Printf("[DOCKER] Removed container %s", containerID[:12])
	return nil
}
