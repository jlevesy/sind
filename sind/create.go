package sind

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Errors.
var (
	ErrEmptyClusterName     = errors.New("empty cluster name")
	ErrEmptyNetworkName     = errors.New("empty network name")
	ErrInvalidManagersCount = errors.New("invalid manager count, must be >= 1")
	ErrInvalidWorkerCount   = errors.New("invalid worker count, must be >= 0")
	ErrPrimaryNodeNotBound  = errors.New("primary node is not exposing docker daemon port")
)

const (
	dockerDINDimage        = "docker:dind"
	defaultSwarmListenAddr = "0.0.0.0:2377"
)

// CreateClusterParams are args to pass to CreateCluster.
type CreateClusterParams struct {
	ClusterName string
	NetworkName string

	Managers int
	Workers  int

	ImageName    string
	PortBindings map[string]string
}

func (n *CreateClusterParams) validate() error {
	if n.ClusterName == "" {
		return ErrEmptyClusterName
	}

	if n.NetworkName == "" {
		return ErrEmptyNetworkName
	}

	if n.Managers < 1 {
		return ErrInvalidManagersCount
	}

	if n.Workers < 0 {
		return ErrInvalidWorkerCount
	}

	return nil
}

func (n *CreateClusterParams) managersToRun() int {
	return n.Managers - 1
}

func (n *CreateClusterParams) imageName() string {
	if n.ImageName != "" {
		return n.ImageName
	}

	return dockerDINDimage
}

func (n *CreateClusterParams) portBindings() (nat.PortMap, error) {
	res := nat.PortMap{}

	for hostPort, rawContainerPort := range n.PortBindings {
		proto, port := nat.SplitProtoPort(rawContainerPort)
		parsedPort, err := nat.NewPort(proto, port)
		if err != nil {
			return nil, err
		}
		res[parsedPort] = []nat.PortBinding{{HostPort: hostPort}}
	}

	return res, nil
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

	net, err := hostClient.NetworkCreate(
		ctx,
		params.NetworkName,
		types.NetworkCreate{
			Labels: map[string]string{
				clusterNameLabel: params.ClusterName,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create cluster network: %v", err)
	}

	pb, err := params.portBindings()
	if err != nil {
		return nil, fmt.Errorf("unable to define port bindings: %v", err)
	}

	primaryNodeCID, err := runContainer(
		ctx,
		hostClient,
		&container.Config{
			Image: params.imageName(),
			Labels: map[string]string{
				clusterNameLabel: params.ClusterName,
				clusterRoleLabel: primaryNode,
			},
		},
		&container.HostConfig{
			Privileged:      true,
			PublishAllPorts: true,
			PortBindings:    pb,
		},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create the primary node: %v", err)
	}

	primaryNodeInfo, err := hostClient.ContainerInspect(ctx, primaryNodeCID)
	if err != nil {
		return nil, fmt.Errorf("unable to get the primary node informations: %v", err)
	}

	swarmPort, err := swarmPort(primaryNodeInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to get the remote docker daemon port: %v", err)
	}

	swarmHost, err := swarmHost(hostClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get the remote docker daemon host: %v", err)
	}

	managerNodeCIDs, err := runContainers(
		ctx,
		hostClient,
		params.managersToRun(),
		&container.Config{
			Image: params.imageName(),
			Labels: map[string]string{
				clusterNameLabel: params.ClusterName,
				clusterRoleLabel: managerNode,
			},
		},
		&container.HostConfig{Privileged: true},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create manager nodes: %v", err)
	}

	workerNodeCIDs, err := runContainers(
		ctx,
		hostClient,
		params.Workers,
		&container.Config{
			Image: params.imageName(),
			Labels: map[string]string{
				clusterNameLabel: params.ClusterName,
				clusterRoleLabel: workerNode,
			},
		},
		&container.HostConfig{Privileged: true},
		networkConfig(params, net.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create worker nodes: %v", err)
	}

	swarmClient, err := docker.NewClientWithOpts(
		docker.WithHost(fmt.Sprintf("tcp://%s:%s", swarmHost, swarmPort)),
		docker.WithVersion("1.39"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create swarm client: %v", err)
	}

	if err = waitDaemonReady(ctx, swarmClient); err != nil {
		return nil, fmt.Errorf("unable to connect to the swarm cluster: %v", err)
	}

	if _, err = swarmClient.SwarmInit(ctx, swarm.InitRequest{ListenAddr: defaultSwarmListenAddr}); err != nil {
		return nil, fmt.Errorf("unable to init the swarm: %v", err)
	}

	swarmInfo, err := swarmClient.SwarmInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to collect join tokens: %v", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(managerNodeCIDs) + len(workerNodeCIDs))
	managerAddr := fmt.Sprintf("%s:2377", primaryNodeCID[0:12])
	for _, cid := range managerNodeCIDs {
		go func(cid string) {
			defer wg.Done()
			execContainer(
				ctx,
				hostClient,
				cid,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					swarmInfo.JoinTokens.Manager,
					managerAddr,
				},
			)
		}(cid)
	}

	for _, cid := range workerNodeCIDs {
		go func(cid string) {
			defer wg.Done()
			execContainer(
				ctx,
				hostClient,
				cid,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					swarmInfo.JoinTokens.Worker,
					managerAddr,
				},
			)
		}(cid)
	}

	wg.Wait()

	return &Cluster{
		Name: params.ClusterName,
		Cluster: Swarm{
			Host: swarmHost,
			Port: swarmPort,
		},
		Host: Docker{
			Host: hostClient.DaemonHost(),
			// TODO support TLS information ?
		},
	}, nil
}

func waitDaemonReady(ctx context.Context, client *docker.Client) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := client.Ping(ctx)
			if err != nil {
				continue
			}

			return nil
		case <-ctx.Done():
			return ctx.Err()
		}

	}
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

func swarmPort(container types.ContainerJSON) (string, error) {
	boundsPorts, ok := container.NetworkSettings.Ports["2375/tcp"]
	if !ok {
		return "", ErrPrimaryNodeNotBound
	}

	if len(boundsPorts) == 0 {
		return "", ErrPrimaryNodeNotBound
	}

	return boundsPorts[0].HostPort, nil
}

func swarmHost(client *docker.Client) (string, error) {
	hostURL, err := url.Parse(client.DaemonHost())
	if err != nil {
		return "", err
	}

	// If it's unix, the bound ports are going to be exposed on localhost
	if hostURL.Scheme == "unix" {
		return "localhost", nil
	}

	return strings.Split(hostURL.Host, ":")[0], nil
}
