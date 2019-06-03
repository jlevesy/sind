package sind

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// ClusterStatus represents the current state of a cluster.
type ClusterStatus struct {
	Name string

	Managers        uint16
	ManagersRunning uint16
	Workers         uint16
	WorkersRunning  uint16

	Nodes []types.Container
}

// InspectCluster returns current status for a given cluster.
// It returns nil,nil if the cluster is not found on the configured docker host.
func InspectCluster(ctx context.Context, hostClient internal.ContainerLister, clusterName string) (*ClusterStatus, error) {
	nodes, err := hostClient.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", internal.ClusterLabel(clusterName))),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get node list: %v", err)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	result := &ClusterStatus{Name: clusterName, Nodes: nodes}

	for _, node := range nodes {
		role, ok := node.Labels[internal.NodeRoleLabel]
		if !ok {
			return nil, fmt.Errorf("node %q has no role label", node.ID)
		}

		if role == internal.NodeRoleManager ||
			role == internal.NodeRolePrimary {

			result.Managers++
			if node.State == "running" {
				result.ManagersRunning++
			}

			continue
		}

		result.Workers++
		if node.State == "running" {
			result.WorkersRunning++
		}
	}

	return result, nil
}
