package sind

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// DeleteCluster will remove all ressources related to a sind cluster according to its name.
func DeleteCluster(ctx context.Context, client *docker.Client, clusterName string) error {
	if err := internal.DeleteNodes(ctx, client, clusterName); err != nil {
		return fmt.Errorf("unable to delete nodes: %v", err)
	}

	if err := internal.DeleteNetwork(ctx, client, clusterName); err != nil {
		return fmt.Errorf("unable to delete network: %v", err)
	}

	return nil
}
