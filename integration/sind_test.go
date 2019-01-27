package integration

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/jlevesy/go-sind/sind"

	"github.com/docker/docker/api/types"
)

func TestSindCanCreateACluster(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Log("Creating cluster")
	params := sind.CreateClusterParams{ClusterName: "test_swarm", NetworkName: "test_swarm", Managers: 3, Workers: 4}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}

	defer func() {
		if err = cluster.Delete(ctx); err != nil {
			t.Fatalf("unable to delete cluster: %v", err)
		}
	}()

	swarmClient, err := cluster.Cluster.Client()
	if err != nil {
		t.Fatalf("unable to get swarm client: %v", err)
	}

	t.Log("Getting cluster informations")

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
	t.Parallel()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		t.Log("Creating cluster n°", i)
		params := sind.CreateClusterParams{
			ClusterName: fmt.Sprintf("foo_%d", i),
			NetworkName: fmt.Sprintf("test_swarm_%d", i),
			Managers:    1,
			Workers:     1,
		}
		cluster, err := sind.CreateCluster(ctx, params)
		if err != nil {
			t.Fatalf("unable to create cluster: %v", err)
		}

		defer func() {
			if err = cluster.Delete(ctx); err != nil {
				t.Fatalf("unable to delete cluster: %v", err)
			}
		}()
	}
}

func TestSindCanPushAnImageToCluster(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tag := "alpine:latest"

	params := sind.CreateClusterParams{ClusterName: "test", NetworkName: "test_swarm", Managers: 1}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}
	defer func() {
		if err = cluster.Delete(ctx); err != nil {
			t.Fatalf("unable to delete cluster: %v", err)
		}
	}()

	hostClient, err := cluster.Host.Client()
	if err != nil {
		t.Fatalf("unable to get a docker client: %v", err)
	}

	out, err := hostClient.ImagePull(ctx, tag, types.ImagePullOptions{})
	if err != nil {
		t.Fatalf("unable to pull the %s image: %v", tag, err)
	}
	defer out.Close()

	if _, err := io.Copy(ioutil.Discard, out); err != nil {
		t.Fatalf("unable to pull the %s image: %v", tag, err)
	}

	if err = cluster.PushImage(ctx, []string{tag}); err != nil {
		t.Fatalf("unable to deploy the %s image to the cluster: %v", tag, err)
	}
}
