package sind

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/sync/errgroup"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// Errors
var (
	ErrNetworkNotFound  = errors.New("network not found")
	ErrNetworkNotUnique = errors.New("network not unique")
)

// Delete will delete the cluster
func (c *Cluster) Delete(ctx context.Context) error {
	// deleteContainers
	if err := c.deleteContainers(ctx); err != nil {
		return fmt.Errorf("unable to delete cluster containers: %v", err)
	}

	if err := c.deleteNetwork(ctx); err != nil {
		return fmt.Errorf("unable to delete cluster network: %v", err)
	}

	return nil
}

func (c *Cluster) deleteContainers(ctx context.Context) error {
	client, err := c.Host.Client()
	if err != nil {
		return fmt.Errorf("unable to get docker client: %v", err)
	}

	containers, err := c.ContainerList(ctx)
	if err != nil {
		return fmt.Errorf("unable to get container list: %v", err)
	}

	var errg errgroup.Group
	for _, container := range containers {
		cid := container.ID
		errg.Go(func() error {
			return client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{Force: true})
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("unable to remove a container: %v", err)
	}

	return nil
}

func (c *Cluster) deleteNetwork(ctx context.Context) error {
	client, err := c.Host.Client()
	if err != nil {
		return fmt.Errorf("unable to get docker client: %v", err)
	}

	networks, err := client.NetworkList(
		ctx,
		types.NetworkListOptions{
			Filters: filters.NewArgs(filters.Arg("label", c.clusterLabel())),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to list cluster networks: %v", err)
	}
	if len(networks) == 0 {
		return ErrNetworkNotFound
	}
	var errg errgroup.Group
	for _, network := range networks {
		netID := network.ID
		errg.Go(func() error {
			return client.NetworkRemove(ctx, netID)
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("unable to delete a network: %v", err)
	}

	return nil
}
