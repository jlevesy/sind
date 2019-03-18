package test

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/jlevesy/sind/pkg/sind"
)

func TestSindCanPushAnImageToCluster(t *testing.T) {
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
		t.Fatalf("unable to push the %s image to the cluster: %v", tag, err)
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
