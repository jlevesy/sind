package internal

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/golang/sync/errgroup"
)

// NetworkConfig represents possible configuration for sind network.
type NetworkConfig struct {
	Name        string
	ClusterName string
	Subnet      string
	Labels      map[string]string
}

type networkCreator interface {
	NetworkCreate(context.Context, string, types.NetworkCreate) (types.NetworkCreateResponse, error)
}

// PickSubnet returns a subnet to use for the container network.
func PickSubnet() (*net.IPNet, error) {
	rand.Seed(time.Now().UnixNano())
	_, res, err := net.ParseCIDR(fmt.Sprintf("10.0.%d.0/24", rand.Intn(256)))

	return res, err
}

// CreateNetwork creates network according to given network config.
func CreateNetwork(ctx context.Context, client networkCreator, cfg NetworkConfig) (types.NetworkCreateResponse, error) {
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}

	cfg.Labels[ClusterNameLabel] = cfg.ClusterName

	return client.NetworkCreate(
		ctx,
		cfg.Name,
		types.NetworkCreate{
			IPAM: &network.IPAM{
				Config: []network.IPAMConfig{
					{Subnet: cfg.Subnet},
				},
			},
			Labels: cfg.Labels,
		},
	)
}

type networkLister interface {
	NetworkList(ctx context.Context, opts types.NetworkListOptions) ([]types.NetworkResource, error)
}

// ListNetworks returns all the networks related to a cluster.
func ListNetworks(ctx context.Context, hostClient networkLister, clusterName string) ([]types.NetworkResource, error) {
	return hostClient.NetworkList(
		ctx,
		types.NetworkListOptions{
			Filters: filters.NewArgs(filters.Arg("label", ClusterLabel(clusterName))),
		},
	)
}

type networkRemover interface {
	NetworkRemove(ctx context.Context, networkID string) error
}

// DeleteNetworks deletes all given networks.
func DeleteNetworks(ctx context.Context, hostClient networkRemover, networks []types.NetworkResource) error {
	errg, groupCtx := errgroup.WithContext(ctx)

	for _, network := range networks {
		netID := network.ID

		errg.Go(func() error {
			return hostClient.NetworkRemove(groupCtx, netID)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("unable to delete a network: %v", err)
	}

	return nil
}
