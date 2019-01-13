package integration

import (
	"context"
	"testing"

	"github.com/jlevesy/go-sind/sind"
)

func TestSindCanCreateACluster(t *testing.T) {
	ctx := context.Background()
	params := sind.CreateClusterParams{ClusterName: "test_swarm", NetworkName: "test_swarm", Managers: 3, Workers: 4}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}

	defer cluster.Delete(ctx)

	swarmClient, err := cluster.Cluster.Client()
	if err != nil {
		t.Fatalf("unable to get swarm client: %v", err)
	}

	info, err := swarmClient.Info(ctx)
	if err != nil {
		t.Fatalf("unable to get swarm info: %v", err)
	}

	if !info.Swarm.ControlAvailable {
		t.Error("expected controlled node to be a manager")
	}

	if info.Swarm.Managers != params.Managers {
		t.Errorf("wrong number of managers created: expected %d, got %d", params.Managers, info.Swarm.Managers)
	}

	if info.Swarm.Nodes-info.Swarm.Managers != params.Workers {
		t.Errorf("wrong number of workers created: expected %d, got %d", params.Workers, info.Swarm.Nodes-info.Swarm.Managers)
	}
}

func TestSindCanCreateMultipleClusters(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		go func() {
			params := sind.CreateClusterParams{ClusterName: "foo", NetworkName: "test_swarm", Managers: 3, Workers: 3}
			cluster, err := sind.CreateCluster(ctx, params)
			if err != nil {
				t.Fatalf("unable to create cluster: %v", err)
			}

			defer cluster.Delete(ctx)
		}()
	}
}
