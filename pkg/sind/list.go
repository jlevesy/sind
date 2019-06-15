package sind

import (
	"context"
	"fmt"

	"github.com/jlevesy/sind/pkg/sind/internal"
)

// ListClusters list the clusters
func ListClusters(ctx context.Context, hostClient internal.ContainerLister) ([]ClusterStatus, error) {
	primaryNodes, err := internal.ListPrimaryContainers(ctx, hostClient)
	if err != nil {
		return nil, err
	}

	result := make([]ClusterStatus, 0, len(primaryNodes))

	for _, node := range primaryNodes {
		clusterName, ok := node.Labels[internal.ClusterNameLabel]
		if !ok {
			return nil, fmt.Errorf("Node %q has no cluster name", node.ID)
		}

		status, err := InspectCluster(ctx, hostClient, clusterName)
		if err != nil {
			return nil, err
		}

		if status == nil {
			return nil, fmt.Errorf("unknown cluster %q", clusterName)
		}

		result = append(result, *status)
	}

	return result, nil
}
