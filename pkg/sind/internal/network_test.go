package internal

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
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
	}{
		{
			desc: "with a nil valuated label map",
			cfg: NetworkConfig{
				Name:        "hello",
				ClusterName: "toto",
			},
			expectedOpts: types.NetworkCreate{
				Labels: map[string]string{
					clusterNameLabel: "toto",
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
					clusterNameLabel: "toto",
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
					clusterNameLabel: "toto",
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
