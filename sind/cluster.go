package sind

import (
	docker "github.com/docker/docker/client"
)

// Cluster is an instance of a swarm cluster.
type Cluster struct {
	PrimaryNodeHost string
	PrimaryNodePort string
	NetworkID       string

	HostClient  *docker.Client
	SwarmClient *docker.Client

	primaryNodeCID  string
	managerNodeCIDs []string
	workerNodeCIDs  []string
}

func (c *Cluster) containerIDs() []string {
	managers := append(c.managerNodeCIDs, c.primaryNodeCID)
	return append(managers, c.workerNodeCIDs...)
}
