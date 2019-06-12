package test

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSindCanStopAndStartACluster(t *testing.T) {
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

	require.NoError(t, sind.StopCluster(ctx, hostClient, params.ClusterName))

	clusterInfos, err := sind.InspectCluster(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	for _, node := range clusterInfos.Nodes {
		assert.Equal(t, "exited", node.State)
	}

	require.NoError(t, sind.StartCluster(ctx, hostClient, params.ClusterName))

	clusterInfos, err = sind.InspectCluster(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	for _, node := range clusterInfos.Nodes {
		assert.Equal(t, "running", node.State)
	}

	swarmHost, err := sind.ClusterHost(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	swarmClient, err := docker.NewClientWithOpts(docker.WithHost(swarmHost), docker.WithVersion("1.39"))
	require.NoError(t, err)

	var info types.Info

	require.NoError(t, retry(10, time.Second, func() error { info, err = swarmClient.Info(ctx); return err }))

	require.True(t, info.Swarm.ControlAvailable)

	assert.EqualValues(t, params.Managers, info.Swarm.Managers)
	assert.EqualValues(t, params.Workers, info.Swarm.Nodes-info.Swarm.Managers)
}
