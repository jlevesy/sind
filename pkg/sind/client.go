package sind

import (
	"context"
	"fmt"
	"net"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind/internal"
)

// ClusterClient configures the docker client to communicate with the swarm primary node.
func ClusterClient(ctx context.Context, hostClient *docker.Client, clusterName string) func(*docker.Client) error {
	return func(client *docker.Client) error {
		primaryNode, err := internal.PrimaryContainer(ctx, hostClient, clusterName)
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

		return docker.WithHost("tcp://" + net.JoinHostPort(swarmHost, fmt.Sprintf("%d", swarmPort)))(client)
	}
}
