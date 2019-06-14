package internal

import (
	"context"
	"errors"
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
			Subnet:      "10.0.117.0/24",
		},
		expectedOpts: types.NetworkCreate{
			IPAM: &network.IPAM{
				Config: []network.IPAMConfig{
					{Subnet: "10.0.117.0/24"},
				},
			},
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
				Subnet:      "10.0.117.0/24",
			},
			expectedOpts: types.NetworkCreate{
				IPAM: &network.IPAM{
					Config: []network.IPAMConfig{
						{Subnet: "10.0.117.0/24"},
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

type networkListerMock func(ctx context.Context, opts types.NetworkListOptions) ([]types.NetworkResource, error)

func (n networkListerMock) NetworkList(ctx context.Context, opts types.NetworkListOptions) ([]types.NetworkResource, error) {
	return n(ctx, opts)
}

func TestNetworkList(t *testing.T) {
	testCases := []struct {
		desc         string
		listError    error
		networks     []types.NetworkResource
		expectsError bool
	}{
		{
			desc: "without error",
			networks: []types.NetworkResource{
				{ID: "Foo"},
			},
			listError:    nil,
			expectsError: false,
		},
		{
			desc: "with error",
			networks: []types.NetworkResource{
				{ID: "Foo"},
			},
			listError:    errors.New("foo"),
			expectsError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var sentOpts types.NetworkListOptions
			client := networkListerMock(func(ctx context.Context, opts types.NetworkListOptions) ([]types.NetworkResource, error) {

				sentOpts = opts
				return test.networks, test.listError
			})
			clusterName := "test"

			res, err := ListNetworks(ctx, client, clusterName)
			if test.expectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.networks, res)
			assert.True(t, sentOpts.Filters.ExactMatch(ClusterNameLabel, clusterName))
		})
	}
}

type networkRemoverMock func(ctx context.Context, networkID string) error

func (n networkRemoverMock) NetworkRemove(ctx context.Context, networkID string) error {
	return n(ctx, networkID)
}

func TestDeleteNetwork(t *testing.T) {
	ctx := context.Background()
	networks := []types.NetworkResource{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
	}

	networkRemoved := make(chan string, len(networks))

	client := networkRemoverMock(func(ctx context.Context, networkID string) error {
		networkRemoved <- networkID
		return nil
	})

	require.NoError(t, DeleteNetworks(ctx, client, networks))

	// assert that all the networks returned are removed.
	close(networkRemoved)
	var removedNetworks []string
	for netID := range networkRemoved {
		removedNetworks = append(removedNetworks, netID)
	}

	sort.Strings(removedNetworks)
	assert.Equal(t, []string{"a", "b", "c", "d"}, removedNetworks)
}
