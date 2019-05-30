package internal

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

// NetworkConfig represents possible configuration for sind network.
type NetworkConfig struct {
	Name        string
	ClusterName string
	Labels      map[string]string
	Subnet      string
}

func (c *NetworkConfig) ipam() *network.IPAM {
	if c.Subnet == "" {
		return nil
	}

	return &network.IPAM{
		Config: []network.IPAMConfig{
			{Subnet: c.Subnet},
		},
	}
}

type networkCreator interface {
	NetworkCreate(context.Context, string, types.NetworkCreate) (types.NetworkCreateResponse, error)
}

// CreateNetwork creates network according to given network config.
func CreateNetwork(ctx context.Context, client networkCreator, cfg NetworkConfig) (types.NetworkCreateResponse, error) {
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}

	cfg.Labels[clusterNameLabel] = cfg.ClusterName

	return client.NetworkCreate(
		ctx,
		cfg.Name,
		types.NetworkCreate{
			IPAM:   cfg.ipam(),
			Labels: cfg.Labels,
		},
	)
}
