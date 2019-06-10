package sind

import (
	"context"
	"fmt"
	"net"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// ClusterHost returns the host to use in order to commnicate with the swarm cluster.
func ClusterHost(ctx context.Context, hostClient *docker.Client, clusterName string) (string, error) {
	primaryNode, err := internal.PrimaryContainer(ctx, hostClient, clusterName)
	if err != nil {
		return "", fmt.Errorf("unable to get the primary node informations: %v", err)
	}

	swarmPort, err := internal.SwarmPort(*primaryNode)
	if err != nil {
		return "", fmt.Errorf("unable to get the remote docker daemon port: %v", err)
	}

	swarmHost, err := internal.SwarmHost(hostClient)
	if err != nil {
		return "", fmt.Errorf("unable to get the remote docker daemon host: %v", err)
	}

	return "tcp://" + net.JoinHostPort(swarmHost, fmt.Sprintf("%d", swarmPort)), nil
}
