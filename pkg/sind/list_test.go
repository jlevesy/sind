package sind

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jlevesy/sind/pkg/sind/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListClusters(t *testing.T) {
	testCases := []struct {
		desc string

		primaryNodes []types.Container
		clusters     map[string][]types.Container

		primaryNodesError error
		clusterListError  error
		expectsError      bool
	}{
		{
			desc:         "with no primary nodes",
			primaryNodes: []types.Container{},
		},
		{
			desc:         "with primary node with a missing cluster name",
			expectsError: true,
			primaryNodes: []types.Container{
				{
					ID:     "foo",
					Labels: map[string]string{},
				},
			},
		},
		{
			desc:              "with a primary list error",
			expectsError:      true,
			primaryNodesError: errors.New("nope"),
		},
		{
			desc:         "with an inspect cluster error",
			expectsError: true,
			primaryNodes: []types.Container{
				{
					ID: "foo",
					Labels: map[string]string{
						internal.ClusterNameLabel: "foo",
					},
				},
			},
			clusterListError: errors.New("nope"),
		},
		{
			desc: "with a valid list of clusters",
			primaryNodes: []types.Container{
				{
					ID: "foo",
					Labels: map[string]string{
						internal.ClusterNameLabel: "foo",
					},
				},
				{
					ID: "bar",
					Labels: map[string]string{
						internal.ClusterNameLabel: "bar",
					},
				},
			},
			clusters: map[string][]types.Container{
				"foo": []types.Container{
					{
						ID: "1",
						Labels: map[string]string{
							internal.NodeRoleLabel: internal.NodeRolePrimary,
						},
					},
				},
				"bar": []types.Container{
					{
						ID: "1",
						Labels: map[string]string{
							internal.NodeRoleLabel: internal.NodeRolePrimary,
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var called bool
			client := internal.ContainerListerMock(func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
				if !called {
					called = true
					return test.primaryNodes, test.primaryNodesError
				}

				clusterNames := opts.Filters.Get("label")
				require.Len(t, clusterNames, 1)

				return test.clusters[strings.TrimPrefix(clusterNames[0], internal.ClusterNameLabel+"=")], test.clusterListError
			})

			clusters, err := ListClusters(ctx, client)
			if test.expectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Len(t, clusters, len(test.clusters))
		})
	}
}
