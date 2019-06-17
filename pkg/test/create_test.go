package test

import (
	"context"
	"fmt"
	"testing"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSindCanCreateACluster(t *testing.T) {
	ctx := context.Background()

	hostClient, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	require.NoError(t, err)

	params := sind.ClusterConfiguration{
		ClusterName: "test_create",
		NetworkName: "test_create",

		Managers: 3,
		Workers:  4,
	}
	require.NoError(t, sind.CreateCluster(ctx, hostClient, params))

	defer func() {
		require.NoError(t, sind.DeleteCluster(ctx, hostClient, params.ClusterName))
	}()

	swarmHost, err := sind.ClusterHost(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	swarmClient, err := docker.NewClientWithOpts(docker.WithHost(swarmHost), docker.WithVersion("1.39"))
	require.NoError(t, err)

	info, err := swarmClient.Info(ctx)
	require.NoError(t, err)

	require.True(t, info.Swarm.ControlAvailable)

	assert.EqualValues(t, params.Managers, info.Swarm.Managers)
	assert.EqualValues(t, params.Workers, info.Swarm.Nodes-info.Swarm.Managers)

	clusterInfos, err := sind.InspectCluster(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	assert.EqualValues(t, params.Managers, clusterInfos.Managers)
	assert.EqualValues(t, params.Managers, clusterInfos.ManagersRunning)

	assert.EqualValues(t, params.Workers, clusterInfos.Workers)
	assert.EqualValues(t, params.Workers, clusterInfos.WorkersRunning)
}

func TestSindCanCreateMultipleClusters(t *testing.T) {
	ctx := context.Background()
	hostClient, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		t.Log("Creating cluster n°", i)
		params := sind.ClusterConfiguration{
			ClusterName: fmt.Sprintf("test_create_parallel_%d", i),
			NetworkName: fmt.Sprintf("test_create_parallel_%d", i),
			Managers:    1,
			Workers:     1,
		}
		require.NoError(t, sind.CreateCluster(ctx, hostClient, params))

		defer func() {
			require.NoError(t, sind.DeleteCluster(ctx, hostClient, params.ClusterName))
		}()
	}
}
