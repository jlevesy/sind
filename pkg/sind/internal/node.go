package internal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/golang/sync/errgroup"
)

// NodesConfig is the configuration for the swarm cluster containers.
type NodesConfig struct {
	ClusterName string
	ImageRef    string

	NetworkID    string
	NetworkName  string
	PortBindings []string

	Managers uint16
	Workers  uint16
}

// NodeIDs carries the IDs of various nodes in the cluster.
type NodeIDs struct {
	Primary  string
	Managers []string
	Workers  []string
}

type containerStarter interface {
	ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(context.Context, string, types.ContainerStartOptions) error
}

// CreateNodes creates the nodes containers of the cluster.
func CreateNodes(ctx context.Context, docker containerStarter, cfg NodesConfig) (*NodeIDs, error) {
	var (
		managerIndex uint16
		workerIndex  uint16

		nodeIPIdentifier uint16 = 1
	)

	primaryCreated := make(chan string, 1)
	managerCreated := make(chan string, cfg.Managers-1)
	workerCreated := make(chan string, cfg.Workers)

	exposedPorts, portBindings, err := nat.ParsePortSpecs(cfg.PortBindings)
	if err != nil {
		return nil, fmt.Errorf("unable to define port bindings: %v", err)
	}

	errg, groupCtx := errgroup.WithContext(ctx)

	// Create the primary node.
	primaryIndex := managerIndex
	primaryIPSuffix := nodeIPIdentifier
	errg.Go(func() error {
		nodeName := fmt.Sprintf("sind-%s-manager-%d", cfg.ClusterName, primaryIndex)
		cID, err := runContainer(
			groupCtx,
			docker,
			&container.Config{
				Hostname:     nodeName,
				Image:        cfg.ImageRef,
				ExposedPorts: nat.PortSet(exposedPorts),
				Labels: map[string]string{
					clusterNameLabel: cfg.ClusterName,
					clusterRoleLabel: nodeRolePrimary,
				},
			},
			&container.HostConfig{
				Privileged:      true,
				PublishAllPorts: true,
				PortBindings:    nat.PortMap(portBindings),
			},
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					cfg.NetworkName: {
						NetworkID: cfg.NetworkID,
						IPAddress: fmt.Sprintf("10.0.117.%d", primaryIPSuffix),
					},
				},
			},
		)

		if err != nil {
			return err
		}
		primaryCreated <- cID
		return nil
	})

	nodeIPIdentifier++
	managerIndex++

	// Create the managers.
	for ; managerIndex < cfg.Managers; managerIndex++ {
		idx := managerIndex
		ipSuffix := nodeIPIdentifier
		errg.Go(func() error {
			nodeName := fmt.Sprintf("sind-%s-manager-%d", cfg.ClusterName, idx)
			cID, err := runContainer(
				groupCtx,
				docker,
				&container.Config{
					Image:    cfg.ImageRef,
					Hostname: nodeName,
					Labels: map[string]string{
						clusterNameLabel: cfg.ClusterName,
						clusterRoleLabel: nodeRoleManager,
					},
				},
				&container.HostConfig{Privileged: true},
				&network.NetworkingConfig{
					EndpointsConfig: map[string]*network.EndpointSettings{
						cfg.NetworkName: {
							NetworkID: cfg.NetworkID,
							IPAddress: fmt.Sprintf("10.0.117.%d", ipSuffix),
						},
					},
				},
			)

			if err != nil {
				return err
			}

			managerCreated <- cID

			return nil
		})
		nodeIPIdentifier++
	}

	// Create the workers.
	for ; workerIndex < cfg.Workers; workerIndex++ {
		idx := workerIndex
		ipSuffix := nodeIPIdentifier
		errg.Go(func() error {
			nodeName := fmt.Sprintf("sind-%s-worker-%d", cfg.ClusterName, idx)
			cID, err := runContainer(
				ctx,
				docker,
				&container.Config{
					Image:    cfg.ImageRef,
					Hostname: nodeName,
					Labels: map[string]string{
						clusterNameLabel: cfg.ClusterName,
						clusterRoleLabel: nodeRoleWorker,
					},
				},
				&container.HostConfig{Privileged: true},
				&network.NetworkingConfig{
					EndpointsConfig: map[string]*network.EndpointSettings{
						cfg.NetworkName: {
							NetworkID: cfg.NetworkID,
							IPAddress: fmt.Sprintf("10.0.117.%d", ipSuffix),
						},
					},
				},
			)

			if err != nil {
				return err
			}

			workerCreated <- cID
			return nil
		})
		nodeIPIdentifier++
	}

	if err = errg.Wait(); err != nil {
		return nil, fmt.Errorf("unable to create the cluster: %v", err)
	}

	close(primaryCreated)
	close(managerCreated)
	close(workerCreated)

	result := NodeIDs{
		Primary: <-primaryCreated,
	}

	for cID := range managerCreated {
		result.Managers = append(result.Managers, cID)
	}

	for cID := range workerCreated {
		result.Workers = append(result.Workers, cID)
	}

	return &result, nil
}

func runContainer(ctx context.Context, client containerStarter, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig) (string, error) {
	resp, err := client.ContainerCreate(
		ctx,
		cConfig,
		hConfig,
		nConfig,
		cConfig.Hostname,
	)
	if err != nil {
		return "", err
	}

	if err = client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}
