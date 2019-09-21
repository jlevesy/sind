package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/golang/sync/errgroup"
)

// ContainerLister is something able to list containers.
type ContainerLister interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
}

// ListPrimaryContainers returns the list of all primary containers known to a docker host.
func ListPrimaryContainers(ctx context.Context, client ContainerLister) ([]types.Container, error) {
	return client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", PrimaryNodeLabel()),
		),
		All: true,
	})
}

// ListContainers returns the lists of containers for given cluster.
func ListContainers(ctx context.Context, docker ContainerLister, clusterName string) ([]types.Container, error) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("label", ClusterLabel(clusterName))),
		All:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get container list: %v", err)
	}

	return containers, nil
}

// PrimaryContainer returns the primary container of given cluster.
func PrimaryContainer(ctx context.Context, docker ContainerLister, clusterName string) (*types.Container, error) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", ClusterLabel(clusterName)),
			filters.Arg("label", PrimaryNodeLabel()),
		),
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %v", err)
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("primary container for cluster %q not found", clusterName)
	}

	if len(containers) > 1 {
		return nil, fmt.Errorf("primary container for cluster %q is not unique", clusterName)
	}

	return &containers[0], nil
}

type containerRemover interface {
	ContainerRemove(ctx context.Context, containerID string, opts types.ContainerRemoveOptions) error
}

// RemoveContainers removes all given containers concurrently.
func RemoveContainers(ctx context.Context, hostClient containerRemover, containers []types.Container) error {
	errg, groupCtx := errgroup.WithContext(ctx)
	for _, container := range containers {
		cid := container.ID
		errg.Go(func() error {
			return hostClient.ContainerRemove(groupCtx,
				cid,
				types.ContainerRemoveOptions{
					Force:         true,
					RemoveVolumes: true,
				},
			)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("failed to remove at least one container: %v", err)
	}

	return nil
}

type containerStarter interface {
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error
}

// StartContainers starts all given containers concurrently.
func StartContainers(ctx context.Context, hostClient containerStarter, containers []types.Container) error {
	errg, groupCtx := errgroup.WithContext(ctx)

	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return hostClient.ContainerStart(groupCtx, cID, types.ContainerStartOptions{})
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("failed to start at least one container: %v", err)
	}

	return nil
}

type containerStopper interface {
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
}

// StopContainers stops all given containers concurrently.
func StopContainers(ctx context.Context, hostClient containerStopper, containers []types.Container) error {
	errg, groupCtx := errgroup.WithContext(ctx)

	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return hostClient.ContainerStop(groupCtx, cID, nil)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("failed to stop at least one container: %v", err)
	}

	return nil
}

type containerContentCopier interface {
	CopyToContainer(context.Context, string, string, io.Reader, types.CopyToContainerOptions) error
}

// CopyToContainers copy content at path to given containers.
func CopyToContainers(ctx context.Context, hostClient containerContentCopier, containers []types.Container, contentPath, destPath string) error {
	errg, groupCtx := errgroup.WithContext(ctx)
	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			file, err := os.Open(contentPath)
			if err != nil {
				return fmt.Errorf("unable to open content: %v", err)
			}

			defer file.Close()

			if err := hostClient.CopyToContainer(groupCtx, cID, destPath, file, types.CopyToContainerOptions{}); err != nil {
				return fmt.Errorf("unable to copy the content to container %q: %v", cID, err)
			}

			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("unable to deploy the image to host: %v", err)
	}

	return nil
}

type executor interface {
	ContainerExecCreate(context.Context, string, types.ExecConfig) (types.IDResponse, error)
	ContainerExecStart(context.Context, string, types.ExecStartCheck) error
}

// ExecContainers execute given command to given containers
func ExecContainers(ctx context.Context, hostClient executor, containers []types.Container, cmd []string) error {
	errg, groupCtx := errgroup.WithContext(ctx)
	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return execContainer(groupCtx, hostClient, cID, cmd)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("unable to exec command %v: %v", cmd, err)
	}

	return nil
}

func execContainer(ctx context.Context, client executor, cID string, cmd []string) error {
	exec, err := client.ContainerExecCreate(
		ctx,
		cID,
		types.ExecConfig{
			Cmd:          cmd,
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		return err
	}

	if err := client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{}); err != nil {
		return err
	}

	return nil
}
