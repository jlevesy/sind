package sind

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("label", c.clusterLabel())),
	})
	if err != nil {
		return fmt.Errorf("unable to get container list: %v", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(containers))

	for _, container := range containers {
		go func(cid string) {
			defer wg.Done()
			client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{Force: true})
		}(container.ID)
	}

	wg.Wait()

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
	if len(networks) > 1 {
		return ErrNetworkNotUnique
	}

	if err = client.NetworkRemove(ctx, networks[0].ID); err != nil {
		return fmt.Errorf("unable to delete cluster network: %v", err)
	}

	return nil
}
