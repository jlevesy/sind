package test

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSindCanPushAnImageToClusterFromRefs(t *testing.T) {
	ctx := context.Background()
	tag := "alpine:latest"

	hostClient, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	require.NoError(t, err)

	params := sind.ClusterConfiguration{
		ClusterName: "test_push",
		NetworkName: "test_push",

		Managers: 1,
		Workers:  2,
	}

	require.NoError(t, sind.CreateCluster(ctx, hostClient, params))

	defer func() {
		require.NoError(t, sind.DeleteCluster(ctx, hostClient, params.ClusterName))
	}()

	out, err := hostClient.ImagePull(ctx, tag, types.ImagePullOptions{})
	require.NoError(t, err)

	defer out.Close()

	_, err = io.Copy(ioutil.Discard, out)
	require.NoError(t, err)

	require.NoError(t, sind.PushImageRefs(ctx, hostClient, params.ClusterName, []string{tag}))

	swarmHost, err := sind.ClusterHost(ctx, hostClient, params.ClusterName)
	require.NoError(t, err)

	swarmClient, err := docker.NewClientWithOpts(docker.WithHost(swarmHost), docker.WithAPIVersionNegotiation())
	require.NoError(t, err)

	imgs, err := swarmClient.ImageList(
		ctx,
		types.ImageListOptions{Filters: filters.NewArgs(filters.Arg("reference", tag))},
	)
	require.NoError(t, err)
	require.Len(t, imgs, 1)

	img := imgs[0]

	assert.Equal(t, img.RepoTags[0], tag)
}
