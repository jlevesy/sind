package sind

import (
	"fmt"

	docker "github.com/docker/docker/client"
)

const (
	clusterNameLabel = "com.sind.cluster.name"
	clusterRoleLabel = "com.sind.cluster.role"

	primaryNode = "primary"
	managerNode = "master"
	workerNode  = "worker"
)

func primaryNodeLabel() string {
	return fmt.Sprintf("%s=%s", clusterRoleLabel, primaryNode)
}

// Cluster carries the information necessary to connect to a swarm cluster.
type Cluster struct {
	Name string
	Host string
}

func (c *Cluster) clusterLabel() string {
	return fmt.Sprintf("%s=%s", clusterNameLabel, c.Name)
}

// HostClient returns a docker client, configured to communicate with the host dameon.
func (c *Cluster) HostClient() (*docker.Client, error) {
	client, err := docker.NewClientWithOpts(
		docker.WithHost(c.Host),
		docker.WithVersion("1.39"),
	)

	if err != nil {
		return nil, err
	}

	return client, nil
}
