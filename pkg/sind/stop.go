package sind

import (
	"context"
	"fmt"

	"github.com/golang/sync/errgroup"
)

// Stop stops all cluster containers.
func (c *Cluster) Stop(ctx context.Context) error {
	hostClient, err := c.Host.Client()
	if err != nil {
		return fmt.Errorf("unable to get host client: %v", err)
	}

	containers, err := c.ContainerList(ctx)
	if err != nil {
		return fmt.Errorf("unable to get container list %v", err)
	}

	errg, groupCtx := errgroup.WithContext(ctx)

	for _, container := range containers {
		cID := container.ID
		errg.Go(func() error {
			return hostClient.ContainerStop(groupCtx, cID, nil)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("unable to stop cluster: %v", err)
	}

	return nil
}
