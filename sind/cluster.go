package sind

import (
	docker "github.com/docker/docker/client"
)

// Cluster is an instance of a swarm cluster.
type Cluster struct {
	networkID string

	primaryNodeCID string
	masterNodeCIDs []string
	workerNodeCIDs []string

	hostClient  *docker.Client
	swarmClient *docker.Client
}

// Swarm returns the docker client setupd to contact the swarm.
func (c *Cluster) Swarm() *docker.Client {
	return c.swarmClient
}

func (c *Cluster) containerIDs() []string {
	masters := append(c.masterNodeCIDs, c.primaryNodeCID)
	return append(masters, c.workerNodeCIDs...)
}
