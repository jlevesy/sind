package test

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSindCanPushAnImageToCluster(t *testing.T) {
	ctx := context.Background()
	tag := "alpine:latest"

	params := sind.CreateClusterParams{ClusterName: "test_push", NetworkName: "test_push", Managers: 1}
	cluster, err := sind.CreateCluster(ctx, params)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, cluster.Delete(ctx))
	}()

	hostClient, err := cluster.Host.Client()
	require.NoError(t, err)

	out, err := hostClient.ImagePull(ctx, tag, types.ImagePullOptions{})
	require.NoError(t, err)
	defer out.Close()

	_, err = io.Copy(ioutil.Discard, out)
	require.NoError(t, err)

	require.NoError(t, cluster.PushImage(ctx, []string{tag}))

	swarmClient, err := cluster.Cluster.Client()
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
