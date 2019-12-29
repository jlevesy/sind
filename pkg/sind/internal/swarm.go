package internal

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/golang/sync/errgroup"
)

const (
	dockerDaemonPort = 2375
	swarmGossipPort  = 2377
)

// SwarmDefaultListenAddress returns the defautl join address for the primary container.
func SwarmDefaultListenAddress() string {
	return net.JoinHostPort("0.0.0.0", strconv.Itoa(swarmGossipPort))
}

// SwarmPort returns the port to use to communicate with the swarm cluster on given primary container.
func SwarmPort(container types.Container) (uint16, error) {
	var swarmPort *types.Port

	for _, port := range container.Ports {
		if port.PrivatePort != dockerDaemonPort {
			continue
		}

		swarmPort = &port

		break
	}

	if swarmPort == nil {
		return 0, fmt.Errorf("container does not export port %d", dockerDaemonPort)
	}

	return swarmPort.PublicPort, nil
}

type hoster interface {
	DaemonHost() string
}

// SwarmHost returns the host of swarm cluster according to client.
func SwarmHost(client hoster) (string, error) {
	daemonURL, err := url.Parse(client.DaemonHost())
	if err != nil {
		return "", err
	}

	if daemonURL.Scheme == "unix" {
		return "localhost", nil
	}

	return daemonURL.Host, nil
}

// ClusterParams are the params for the cluster.
type ClusterParams struct {
	IDs NodeIDs

	PrimaryNodeIP    string
	ManagerJoinToken string
	WorkerJoinToken  string
}

// FormCluster make managers and workers to join the primary node.
func FormCluster(ctx context.Context, client executor, params ClusterParams) error {
	errg, groupCtx := errgroup.WithContext(ctx)

	managerAddr := net.JoinHostPort(params.PrimaryNodeIP, strconv.Itoa(swarmGossipPort))

	for _, managerID := range params.IDs.Managers {
		cid := managerID

		errg.Go(func() error {
			return execContainer(
				groupCtx,
				client,
				cid,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					params.ManagerJoinToken,
					managerAddr,
				},
			)
		})
	}

	for _, workerID := range params.IDs.Workers {
		cid := workerID

		errg.Go(func() error {
			return execContainer(
				groupCtx,
				client,
				cid,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					params.WorkerJoinToken,
					managerAddr,
				},
			)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("unable to form the cluster: %v", err)
	}

	return nil
}
