package sind

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// ListClusters list the clusters
func ListClusters(ctx context.Context, hostClient *docker.Client) ([]ClusterStatus, error) {
	primaryNodes, err := internal.ListPrimaryContainers(ctx, hostClient)
	if err != nil {
		return nil, err
	}

	result := make([]ClusterStatus, 0, len(primaryNodes))

	for _, node := range primaryNodes {
		clusterName, ok := node.Labels[internal.ClusterNameLabel]
		if !ok {
			return nil, fmt.Errorf("Node %q has not cluster name", node.ID)
		}

		status, err := InspectCluster(ctx, hostClient, clusterName)
		if err != nil {
			return nil, err
		}

		result = append(result, *status)
	}

	return result, nil
}
