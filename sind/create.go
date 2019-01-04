package sind

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Errors.
var (
	ErrEmptyNetworkName    = errors.New("empty network name")
	ErrInvalidMastersCount = errors.New("invalid master count, must be >= 1")
	ErrInvalidWorkerCount  = errors.New("invalid worker count, must be >= 0")
)

const (
	dockerDINDimage        = "docker:dind"
	dockerdTCPListenPort   = "2375"
	defaultSwarmListenAddr = "0.0.0.0:2377"
)

// CreateClusterParams are args to pass to CreateCluster.
type CreateClusterParams struct {
	NetworkName string

	Masters int
	Workers int

	ImageName    string
	PortBindings map[string]string
}

func (n *CreateClusterParams) validate() error {
	if n.NetworkName == "" {
		return ErrEmptyNetworkName
	}

	if n.Masters < 1 {
		return ErrInvalidMastersCount
	}

	if n.Workers < 0 {
		return ErrInvalidWorkerCount
	}

	return nil
}

func (n *CreateClusterParams) mastersToRun() int {
	return n.Masters - 1
}

func (n *CreateClusterParams) imageName() string {
	if n.ImageName != "" {
		return n.ImageName
	}

	return dockerDINDimage
}

func (n *CreateClusterParams) portBindings() nat.PortMap {
	res := nat.PortMap{}

	res[nat.Port(path.Join(dockerdTCPListenPort, "tcp"))] = []nat.PortBinding{{HostPort: dockerdTCPListenPort}}

	for hostPort, cPort := range n.PortBindings {
		res[nat.Port(cPort)] = []nat.PortBinding{{HostPort: hostPort}}
	}

	return res
}

// CreateCluster creates a new swarm cluster.
func CreateCluster(ctx context.Context, params CreateClusterParams) (*Cluster, error) {
	if err := params.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	hostClient, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %v", err)
	}

	net, err := hostClient.NetworkCreate(ctx, params.NetworkName, types.NetworkCreate{})
	if err != nil {
		return nil, fmt.Errorf("unable to create cluster network: %v", err)
	}

	primaryNodeCID, err := runContainer(
		ctx,
		hostClient,
		&container.Config{Image: params.imageName()},
		&container.HostConfig{
			Privileged:   true,
			PortBindings: params.portBindings(),
		},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create the primary node: %v", err)
	}

	masterNodeCIDs, err := runContainers(
		ctx,
		hostClient,
		params.mastersToRun(),
		&container.Config{Image: params.imageName()},
		&container.HostConfig{Privileged: true},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create master nodes: %v", err)
	}

	workerNodeCIDs, err := runContainers(
		ctx,
		hostClient,
		params.Workers,
		&container.Config{Image: params.imageName()},
		&container.HostConfig{Privileged: true},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create worker nodes: %v", err)
	}

	swarmClient, err := docker.NewClientWithOpts(docker.WithHost("tcp://localhost:2375"), docker.WithVersion("1.39"))
	if err != nil {
		return nil, fmt.Errorf("unable to create swarm client: %v", err)
	}

	if _, err = swarmClient.SwarmInit(ctx, swarm.InitRequest{ListenAddr: defaultSwarmListenAddr}); err != nil {
		return nil, fmt.Errorf("unable to init the swarm: %v", err)
	}

	swarmInfo, err := swarmClient.SwarmInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to collect join tokens: %v", err)
	}

	masterAddr := fmt.Sprintf("%s:2377", primaryNodeCID[0:12])
	// joinMasters
	for _, cid := range masterNodeCIDs {
		go execContainer(
			ctx,
			hostClient,
			cid,
			[]string{
				"docker",
				"swarm",
				"join",
				"--token",
				swarmInfo.JoinTokens.Manager,
				masterAddr,
			},
		)
	}
	// joinWorkers
	for _, cid := range workerNodeCIDs {
		go execContainer(
			ctx,
			hostClient,
			cid,
			[]string{
				"docker",
				"swarm",
				"join",
				"--token",
				swarmInfo.JoinTokens.Worker,
				masterAddr,
			},
		)
	}

	return &Cluster{
		networkID:      net.ID,
		primaryNodeCID: primaryNodeCID,
		masterNodeCIDs: masterNodeCIDs,
		workerNodeCIDs: workerNodeCIDs,
		hostClient:     hostClient,
		swarmClient:    swarmClient,
	}, nil
}

func execContainer(ctx context.Context, client *docker.Client, cID string, cmd []string) error {
	exec, err := client.ContainerExecCreate(ctx, cID, types.ExecConfig{Cmd: cmd, AttachStdout: true, AttachStderr: true})
	if err != nil {
		return err
	}

	return client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
}

func runContainer(ctx context.Context, client *docker.Client, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig) (string, error) {
	resp, err := client.ContainerCreate(
		ctx,
		cConfig,
		hConfig,
		nConfig,
		"",
	)
	if err != nil {
		return "", err
	}

	if err = client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func runContainers(ctx context.Context, client *docker.Client, totalToCreate int, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig) ([]string, error) {
	result := []string{}

	if totalToCreate == 0 {
		return result, nil
	}

	containersCreatedCount := 0

	creationErrors := make(chan error)
	containerCreated := make(chan string, totalToCreate)

	for i := 0; i < totalToCreate; i++ {
		go func() {
			cID, err := runContainer(ctx, client, cConfig, hConfig, nConfig)
			if err != nil {
				creationErrors <- err
			}

			containerCreated <- cID
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-creationErrors:
			return nil, err
		case cid := <-containerCreated:
			result = append(result, cid)
			containersCreatedCount++
			if containersCreatedCount != totalToCreate {
				continue
			}

			return result, nil
		}
	}
}

func networkConfig(params CreateClusterParams, networkID string) *network.NetworkingConfig {
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			params.NetworkName: {
				NetworkID: networkID,
			},
		},
	}
}
