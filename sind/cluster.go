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

// Cluster carries the information necessary to connect to a swarm cluster.
type Cluster struct {
	Name string

	Host    Docker
	Cluster Swarm
}

func (c *Cluster) clusterLabel() string {
	return fmt.Sprintf("%s=%s", clusterNameLabel, c.Name)
}

// Swarm are the informations required to connect to the swarm cluster.
type Swarm struct {
	Host string
	Port string

	client *docker.Client
}

// Client will return a instance of the docker client.
func (s *Swarm) Client() (*docker.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	client, err := docker.NewClientWithOpts(
		docker.WithHost(fmt.Sprintf("tcp://%s:%s", s.Host, s.Port)),
		docker.WithVersion("1.39"),
	)

	if err != nil {
		return nil, err
	}

	s.client = client

	return client, nil
}

// Docker are the informations required ot connect to the host daemon.
type Docker struct {
	Host string

	client *docker.Client
}

// Client will return a instance of the docker client.
func (d *Docker) Client() (*docker.Client, error) {
	if d.client != nil {
		return d.client, nil
	}

	client, err := docker.NewClientWithOpts(
		docker.WithHost(d.Host),
		docker.WithVersion("1.39"),
	)

	if err != nil {
		return nil, err
	}

	d.client = client

	return client, nil
}
