package sind

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// DeleteCluster will remove all ressources related to a sind cluster according to its name.
func DeleteCluster(ctx context.Context, client *docker.Client, clusterName string) error {
	nodes, err := internal.ListContainers(ctx, client, clusterName)
	if err != nil {
		return fmt.Errorf("unable to list nodes: %v", err)
	}

	nets, err := internal.ListNetworks(ctx, client, clusterName)
	if err != nil {
		return fmt.Errorf("unable to list cluster networks: %v", err)
	}

	if err := internal.RemoveContainers(ctx, client, nodes); err != nil {
		return fmt.Errorf("unable to delete nodes: %v", err)
	}

	if err := internal.DeleteNetworks(ctx, client, nets); err != nil {
		return fmt.Errorf("unable to delete networks: %v", err)
	}

	return nil
}
