package sind

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/docker/docker/api/types/swarm"
	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

const (
	// DefaultNodeImageName is the default image name to use for creating swarm nodes.
	DefaultNodeImageName = "docker:18.09-dind"
)

// ClusterConfiguration are args to pass to CreateCluster.
type ClusterConfiguration struct {
	ClusterName string
	NetworkName string

	Managers uint16
	Workers  uint16

	ImageName    string
	PullImage    bool
	PortBindings []string
}

func (n *ClusterConfiguration) validate() error {
	if n.ClusterName == "" {
		return errors.New("cluster name is required")
	}

	if n.NetworkName == "" {
		return errors.New("network name is required")
	}

	if n.Managers < 1 {
		return errors.New("invalid manager count, must be >= 1")
	}

	return nil
}

func (n *ClusterConfiguration) imageName() string {
	if n.ImageName != "" {
		return n.ImageName
	}

	return DefaultNodeImageName
}

// CreateCluster creates a new swarm cluster.
func CreateCluster(ctx context.Context, hostClient *docker.Client, params ClusterConfiguration) error {
	if err := params.validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	imageExists, err := internal.ImageExists(ctx, hostClient, params.imageName())
	if err != nil {
		return fmt.Errorf("unable to check node image existence: %v", err)
	}

	if params.PullImage || !imageExists {
		if err := internal.PullImage(ctx, hostClient, params.imageName()); err != nil {
			return fmt.Errorf("unable to pull the %s image: %v", params.imageName(), err)
		}
	}

	subnet, err := internal.PickSubnet()
	if err != nil {
		return fmt.Errorf("unable to pick an internal subnet: %v", err)
	}

	networkCfg := internal.NetworkConfig{
		Name:        params.NetworkName,
		ClusterName: params.ClusterName,
		Subnet:      subnet.String(),
	}

	clusterNet, err := internal.CreateNetwork(ctx, hostClient, networkCfg)
	if err != nil {
		return fmt.Errorf("unable to create cluster network: %v", err)
	}

	nodesCfg := internal.NodesConfig{
		ClusterName: params.ClusterName,
		ImageRef:    params.imageName(),

		NetworkID:    clusterNet.ID,
		NetworkName:  params.NetworkName,
		Subnet:       *subnet,
		PortBindings: params.PortBindings,

		Managers: params.Managers,
		Workers:  params.Workers,
	}

	nodecIDs, err := internal.CreateNodes(ctx, hostClient, nodesCfg)
	if err != nil {
		return fmt.Errorf("unable to create nodes: %v", err)
	}

	primaryNode, err := internal.PrimaryContainer(ctx, hostClient, params.ClusterName)
	if err != nil {
		return fmt.Errorf("unable to get the primary node informations: %v", err)
	}

	swarmPort, err := internal.SwarmPort(*primaryNode)
	if err != nil {
		return fmt.Errorf("unable to get the remote docker daemon port: %v", err)
	}

	swarmHost, err := internal.SwarmHost(hostClient)
	if err != nil {
		return fmt.Errorf("unable to get the remote docker daemon host: %v", err)
	}

	swarmClient, err := docker.NewClientWithOpts(
		docker.WithHost("tcp://"+net.JoinHostPort(swarmHost, fmt.Sprintf("%d", swarmPort))),
		docker.WithVersion("1.39"),
	)
	if err != nil {
		return fmt.Errorf("unable to create swarm client: %v", err)
	}

	if err = internal.WaitDaemonReady(ctx, swarmClient); err != nil {
		return fmt.Errorf("unable to contact the primary node daemon: %v", err)
	}

	if _, err = swarmClient.SwarmInit(
		ctx, swarm.InitRequest{ListenAddr: internal.SwarmDefaultListenAddress()}); err != nil {
		return fmt.Errorf("unable to init the swarm: %v", err)
	}

	primaryNodeEndpoint, present := primaryNode.NetworkSettings.Networks[params.NetworkName]
	if !present {
		return fmt.Errorf("primary node is not a member of the cluster network")
	}

	swarmInfo, err := swarmClient.SwarmInspect(ctx)
	if err != nil {
		return fmt.Errorf("unable to collect swarm cluster informations: %v", err)
	}

	clusterConfig := internal.ClusterParams{
		IDs: *nodecIDs,

		PrimaryNodeIP:    primaryNodeEndpoint.IPAddress,
		ManagerJoinToken: swarmInfo.JoinTokens.Manager,
		WorkerJoinToken:  swarmInfo.JoinTokens.Worker,
	}

	if err = internal.FormCluster(ctx, hostClient, clusterConfig); err != nil {
		return fmt.Errorf("unable to form the swarm cluster: %v", err)
	}

	return nil
}
