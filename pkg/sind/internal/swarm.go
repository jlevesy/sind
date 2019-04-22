package internal

import (
	"fmt"

	"github.com/docker/docker/api/types"
)

const (
	dockerDaemonPort = 2375
)

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
