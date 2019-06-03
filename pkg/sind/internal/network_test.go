package internal

import (
	"context"
	"sort"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type networkCreatorMock func(context.Context, string, types.NetworkCreate) (types.NetworkCreateResponse, error)

func (n networkCreatorMock) NetworkCreate(ctx context.Context, name string, opts types.NetworkCreate) (types.NetworkCreateResponse, error) {
	return n(ctx, name, opts)
}

func TestCreateNetwork(t *testing.T) {
	testCases := []struct {
		desc         string
		cfg          NetworkConfig
		expectedOpts types.NetworkCreate
	}{{
		desc: "with a nil valuated label map",
		cfg: NetworkConfig{
			Name:        "hello",
			ClusterName: "toto",
		},
		expectedOpts: types.NetworkCreate{
			Labels: map[string]string{
				ClusterNameLabel: "toto",
			},
		},
	},
		{
			desc: "without subnet",
			cfg: NetworkConfig{
				Name:        "hello",
				Labels:      map[string]string{"foo": "bar"},
				ClusterName: "toto",
			},
			expectedOpts: types.NetworkCreate{
				Labels: map[string]string{
					"foo":            "bar",
					ClusterNameLabel: "toto",
				},
			},
		},
		{
			desc: "with subnet",
			cfg: NetworkConfig{
				Name:        "hello",
				Labels:      map[string]string{"foo": "bar"},
				Subnet:      "10.0.0.1/24",
				ClusterName: "toto",
			},
			expectedOpts: types.NetworkCreate{
				IPAM: &network.IPAM{
					Config: []network.IPAMConfig{
						{Subnet: "10.0.0.1/24"},
					},
				},
				Labels: map[string]string{
					"foo":            "bar",
					ClusterNameLabel: "toto",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var (
				sentName string
				sentOpts types.NetworkCreate
			)
			mock := networkCreatorMock(func(ctx context.Context, name string, opts types.NetworkCreate) (types.NetworkCreateResponse, error) {
				sentName = name
				sentOpts = opts
				return types.NetworkCreateResponse{}, nil
			})
			_, _ = CreateNetwork(ctx, mock, test.cfg)
			assert.Equal(t, test.cfg.Name, sentName)
			assert.Equal(t, test.expectedOpts, sentOpts)
		})
	}
}

type networkDeleterMock struct {
	networkList   func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	networkRemove func(ctx context.Context, networkID string) error
}

func (d networkDeleterMock) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	return d.networkList(ctx, options)
}

func (d networkDeleterMock) NetworkRemove(ctx context.Context, networkID string) error {
	return d.networkRemove(ctx, networkID)
}

func TestDeleteNetwork(t *testing.T) {
	ctx := context.Background()

	var listOpts types.NetworkListOptions
	networks := []types.NetworkResource{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
	}

	clusterName := "foo"
	networkRemoved := make(chan string, len(networks))

	client := networkDeleterMock{
		networkList: func(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
			listOpts = options
			return networks, nil
		},
		networkRemove: func(ctx context.Context, networkID string) error {
			networkRemoved <- networkID
			return nil
		},
	}

	require.NoError(t, DeleteNetwork(ctx, client, clusterName))

	// assert correctness of the filters passed to list network.
	assert.True(t, listOpts.Filters.ExactMatch(ClusterNameLabel, clusterName))

	// assert that all the networks returned are removed.
	close(networkRemoved)
	var removedNetworks []string
	for netID := range networkRemoved {
		removedNetworks = append(removedNetworks, netID)
	}

	sort.Strings(removedNetworks)
	assert.Equal(t, []string{"a", "b", "c", "d"}, removedNetworks)
}
