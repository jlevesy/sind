package integration

import (
	"context"
	"testing"

	"github.com/jlevesy/sind/sind"
)

func TestSindCanCreateACluster(t *testing.T) {
	ctx := context.Background()
	params := sind.CreateClusterParams{NetworkName: "test_swarm", Masters: 3, Workers: 4}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}

	defer cluster.Delete(ctx)

	info, err := cluster.Swarm().Info(ctx)
	if err != nil {
		t.Fatalf("unable to get swarm info: %v", err)
	}

	if !info.Swarm.ControlAvailable {
		t.Error("expected controlled node to be a manager")
	}

	if info.Swarm.Managers != params.Masters {
		t.Errorf("wrong number of managers created: expected %d, got %d", params.Masters, info.Swarm.Managers)
	}

	if info.Swarm.Nodes-info.Swarm.Managers != params.Workers {
		t.Errorf("wrong number of workers created: expected %d, got %d", params.Workers, info.Swarm.Nodes-info.Swarm.Managers)
	}
}
