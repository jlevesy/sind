package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/golang/sync/errgroup"
)

type containerLister interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
}

// ListContainers returns the lists of containers for given cluster.
func ListContainers(ctx context.Context, docker containerLister, clusterName string) ([]types.Container, error) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("label", clusterLabel(clusterName))),
		All:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get container list: %v", err)
	}

	return containers, nil
}

// PrimaryContainer returns the primary container of given cluster.
func PrimaryContainer(ctx context.Context, docker containerLister, clusterName string) (*types.Container, error) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", clusterLabel(clusterName)),
			filters.Arg("label", primaryNodeLabel),
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

type containerRemover interface {
	ContainerRemove(ctx context.Context, containerID string, opts types.ContainerRemoveOptions) error
}

// RemoveContainers removes all given containers concurrently.
func RemoveContainers(ctx context.Context, hostClient containerRemover, containers []types.Container) error {
	errg, groupCtx := errgroup.WithContext(ctx)
	for _, container := range containers {
		cid := container.ID
		errg.Go(func() error {
			return hostClient.ContainerRemove(
				groupCtx,
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
