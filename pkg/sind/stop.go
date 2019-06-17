package sind

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// StopCluster stops all nodes of a cluster.
func StopCluster(ctx context.Context, hostClient *docker.Client, clusterName string) error {
	containers, err := internal.ListContainers(ctx, hostClient, clusterName)
	if err != nil {
		return fmt.Errorf("unable to get container list %v", err)
	}

	return internal.StopContainers(ctx, hostClient, containers)
}
