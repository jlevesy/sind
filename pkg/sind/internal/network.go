package internal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/golang/sync/errgroup"
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

type networkDeleter interface {
	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	NetworkRemove(ctx context.Context, networkID string) error
}

// DeleteNetwork deletes all networks related to a clusterName.
func DeleteNetwork(ctx context.Context, client networkDeleter, clusterName string) error {
	networks, err := client.NetworkList(
		ctx,
		types.NetworkListOptions{
			Filters: filters.NewArgs(filters.Arg("label", clusterLabel(clusterName))),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to list cluster networks: %v", err)
	}

	errg, groupCtx := errgroup.WithContext(ctx)
	for _, network := range networks {
		netID := network.ID
		errg.Go(func() error {
			return client.NetworkRemove(groupCtx, netID)
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("unable to delete a network: %v", err)
	}

	return nil
}
