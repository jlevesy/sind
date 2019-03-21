package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/jlevesy/sind/pkg/sind"
)

func TestSindCanCreateACluster(t *testing.T) {
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

func TestSindCanCreateAClusterWithCustomImage(t *testing.T) {
	ctx := context.Background()
	t.Log("Creating cluster with custom image name")

	params := sind.CreateClusterParams{ClusterName: "test_swarm", NetworkName: "test_swarm", Managers: 3, Workers: 4, ImageName: "docker:dind"}
	cluster, err := sind.CreateCluster(ctx, params)
	if err != nil {
		t.Fatalf("unable to create cluster: %v", err)
	}

	defer func() {
		if err = cluster.Delete(ctx); err != nil {
			t.Fatalf("unable to delete cluster: %v", err)
		}
	}()

	dockerCli, err := cluster.Host.Client()
	if err != nil {
		t.Fatalf("unable to get a docker client to the host: %v", err)
	}

	listFilters := filters.NewArgs(filters.Arg("ancestor", params.ImageName), filters.Arg("status", "running"))
	runningContainers, err := dockerCli.ContainerList(ctx, types.ContainerListOptions{Filters: listFilters})
	if err != nil {
		t.Fatalf("unable to retrieve a list of running containers on the host: %v", err)
	}

	if len(runningContainers) != params.Managers+params.Workers {
		t.Errorf("invalid amount of running containers based on the image %s. Expected: %d, Got: %d.", params.ImageName, params.Managers+params.Workers, len(runningContainers))
	}
}

func TestSindCanCreateMultipleClusters(t *testing.T) {
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

func TestSindCanCreateAClusterWithCustomSubnet(t *testing.T) {
	ctx := context.Background()
	t.Log("Creating cluster with custom subnet")

	params := sind.CreateClusterParams{
		ClusterName:   "test_swarm",
		NetworkName:   "test_swarm",
		Managers:      3,
		Workers:       4,
		NetworkSubnet: "10.7.0.0/24",
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

	dockerClient, err := cluster.Host.Client()
	if err != nil {
		t.Fatalf("unable to get a docker client to the host: %v", err)
	}

	networks, err := dockerClient.NetworkList(
		ctx,
		types.NetworkListOptions{
			Filters: filters.NewArgs(filters.Arg("name", params.NetworkName)),
		},
	)
	if err != nil {
		t.Fatalf("unable to get the network list: %v", err)
	}

	if len(networks) != 1 {
		t.Fatalf("invalid number of networks, expected 1, got %d", len(networks))
	}

	net := networks[0]

	if len(net.IPAM.Config) != 1 {
		t.Fatalf("invalid number of IPAM configs, expected 1, got %d", len(net.IPAM.Config))
	}

	if net.IPAM.Config[0].Subnet != params.NetworkSubnet {
		t.Errorf("invalid IPAM subnet, expected %s, got %s", params.NetworkSubnet, net.IPAM.Config[0].Subnet)
	}
}
