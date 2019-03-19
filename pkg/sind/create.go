package sind

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/golang/sync/errgroup"
)

// Errors.
const (
	ErrEmptyClusterName     = "empty cluster name"
	ErrEmptyNetworkName     = "empty network name"
	ErrInvalidManagersCount = "invalid manager count, must be >= 1"
	ErrInvalidWorkerCount   = "invalid worker count, must be >= 0"
	ErrPrimaryNodeNotBound  = "primary node is not exposing docker daemon port"
)

const (
	defaultSwarmListenAddr = "0.0.0.0:2377"
)

const (
	// DefaultNodeImageName is the default image name to use for creating swarm nodes.
	DefaultNodeImageName = "docker:18.09-dind"
)

// CreateClusterParams are args to pass to CreateCluster.
type CreateClusterParams struct {
	ClusterName string
	NetworkName string

	Managers int
	Workers  int

	ImageName    string
	PortBindings []string
}

func (n *CreateClusterParams) validate() error {
	if n.ClusterName == "" {
		return errors.New(ErrEmptyClusterName)
	}

	if n.NetworkName == "" {
		return errors.New(ErrEmptyNetworkName)
	}

	if n.Managers < 1 {
		return errors.New(ErrInvalidManagersCount)
	}

	if n.Workers < 0 {
		return errors.New(ErrInvalidWorkerCount)
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

	return DefaultNodeImageName
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

	out, err := hostClient.ImagePull(ctx, params.imageName(), types.ImagePullOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to pull the %s image: %v", params.imageName(), err)
	}
	defer out.Close()

	if _, err = io.Copy(ioutil.Discard, out); err != nil {
		return nil, fmt.Errorf("unable to pull the %s image: %v", params.imageName(), err)
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

	exposedPorts, portBindings, err := nat.ParsePortSpecs(params.PortBindings)
	if err != nil {
		return nil, fmt.Errorf("unable to define port bindings: %v", err)
	}

	primaryNodeCID, err := runContainer(
		ctx,
		hostClient,
		&container.Config{
			Image:        params.imageName(),
			ExposedPorts: nat.PortSet(exposedPorts),
			Labels: map[string]string{
				clusterNameLabel: params.ClusterName,
				clusterRoleLabel: primaryNode,
			},
		},
		&container.HostConfig{
			Privileged:      true,
			PublishAllPorts: true,
			PortBindings:    nat.PortMap(portBindings),
		},
		networkConfig(params, net.ID),
		"manager-0",
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
		"manager",
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
		"worker",
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

	var errg errgroup.Group
	managerAddr := fmt.Sprintf("%s:2377", primaryNodeCID[0:12])
	for _, managerID := range managerNodeCIDs {
		cid := managerID
		errg.Go(func() error {
			return execContainer(
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
		})
	}

	for _, workerID := range workerNodeCIDs {
		cid := workerID
		errg.Go(func() error {
			return execContainer(
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
		})
	}

	if err = errg.Wait(); err != nil {
		return nil, fmt.Errorf("unable to build the cluster: %v", err)
	}

	if err = waitClusterReady(ctx, swarmClient, params.Managers, params.Workers); err != nil {
		return nil, fmt.Errorf("unable to check swarm cluste: %v", err)
	}

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

func waitClusterReady(ctx context.Context, client *docker.Client, expectedManagers, expectedWorkers int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nodes, err := client.NodeList(ctx, types.NodeListOptions{})
			if err != nil {
				continue
			}

			managers, workers := countNodesPerRole(nodes)

			if managers != expectedManagers {
				continue
			}

			if workers != expectedWorkers {
				continue
			}

			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func countNodesPerRole(nodes []swarm.Node) (managersCount int, workersCount int) {
	for _, node := range nodes {
		// If the node is not ready then don't count it.
		if node.Status.State != swarm.NodeStateReady {
			continue
		}

		switch node.Spec.Role {
		case swarm.NodeRoleManager:
			managersCount++
		case swarm.NodeRoleWorker:
			workersCount++
		}
	}

	return managersCount, workersCount
}

func execContainer(ctx context.Context, client *docker.Client, cID string, cmd []string) error {
	exec, err := client.ContainerExecCreate(ctx, cID, types.ExecConfig{Cmd: cmd, AttachStdout: true, AttachStderr: true})
	if err != nil {
		return err
	}

	return client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
}

func runContainer(ctx context.Context, client *docker.Client, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig, name string) (string, error) {
	cConfig.Hostname = name
	resp, err := client.ContainerCreate(
		ctx,
		cConfig,
		hConfig,
		nConfig,
		name,
	)
	if err != nil {
		return "", err
	}

	if err = client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func runContainers(ctx context.Context, client *docker.Client, totalToCreate int, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig, namePrefix string) ([]string, error) {
	var result []string

	if totalToCreate == 0 {
		return result, nil
	}

	containersCreatedCount := 0

	creationErrors := make(chan error)
	containerCreated := make(chan string, totalToCreate)

	for i := 0; i < totalToCreate; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i+1)
		go func() {
			cID, err := runContainer(ctx, client, cConfig, hConfig, nConfig, name)
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
		return "", errors.New(ErrPrimaryNodeNotBound)
	}

	if len(boundsPorts) == 0 {
		return "", errors.New(ErrPrimaryNodeNotBound)
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
