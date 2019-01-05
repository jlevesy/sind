package sind

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
)

// Delete will delete the cluster
func (c *Cluster) Delete(ctx context.Context) error {
	// deleteContainers
	c.deleteContainers(ctx)

	// deleteNetwork
	return c.hostClient.NetworkRemove(ctx, c.networkID)
}

func (c *Cluster) deleteContainers(ctx context.Context) {
	containers := c.containerIDs()
	wg := sync.WaitGroup{}
	wg.Add(len(containers))

	for _, cid := range containers {
		go func(cid string) {
			defer wg.Done()
			c.hostClient.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{Force: true})
		}(cid)
	}

	wg.Wait()
}