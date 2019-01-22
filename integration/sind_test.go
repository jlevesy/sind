package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/jlevesy/go-sind/sind"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

		defer cluster.Delete(ctx)
	}
}

func TestSindCanPushAnImageToCluster(t *testing.T) {
	ctx := context.Background()
	tag := "alpine:latest"
	params := sind.CreateClusterParams{ClusterName: "test", NetworkName: "test_swarm", Managers: 1}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}
	defer cluster.Delete(ctx)

	hostClient, err := cluster.Host.Client()
	if err != nil {
		t.Fatalf("unable to get a docker client: %v", err)
	}

	out, err := hostClient.ImagePull(ctx, tag, types.ImagePullOptions{})
	if err != nil {
		t.Fatalf("unable to pull the alpine:latest image: %v", err)
	}
	defer out.Close()

	if err = cluster.PushImage(ctx, []string{tag}); err != nil {
		t.Fatalf("unable to deploy the alpine:latest image to the cluster: %v", err)
	}

	swarmClient, err := cluster.Cluster.Client()
	if err != nil {
		t.Fatalf("unable to get a swarm client: %v", err)
	}

	imgs, err := swarmClient.ImageList(
		ctx,
		types.ImageListOptions{Filters: filters.NewArgs(filters.Arg("reference", tag))},
	)
	if err != nil {
		t.Fatalf("unable to fetch images from swarm cluster: %v", err)
	}

	if len(imgs) != 1 {
		t.Fatalf("expected to have one image deployed to the cluster, got %d", len(imgs))
	}

	img := imgs[0]

	if img.RepoTags[0] != tag {
		t.Fatalf("invalid tag found, expected %s got %s", tag, img.RepoTags[0])
	}
}
