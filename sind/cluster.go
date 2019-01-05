package sind

import (
	docker "github.com/docker/docker/client"
)

// Cluster is an instance of a swarm cluster.
type Cluster struct {
	networkID string

	primaryNodeCID  string
	managerNodeCIDs []string
	workerNodeCIDs  []string

	hostClient  *docker.Client
	swarmClient *docker.Client
}

// Swarm returns the docker client setup to contact the swarm.
func (c *Cluster) Swarm() *docker.Client {
	return c.swarmClient
}

func (c *Cluster) containerIDs() []string {
	managers := append(c.managerNodeCIDs, c.primaryNodeCID)
	return append(managers, c.workerNodeCIDs...)
}
