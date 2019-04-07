package sind

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

// ContainerList returns the lists of containers.
func (c *Cluster) ContainerList(ctx context.Context) ([]types.Container, error) {
	client, err := c.Host.Client()
	if err != nil {
		return nil, fmt.Errorf("unable to get docker client: %v", err)
	}

	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("label", c.clusterLabel())),
		All:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get container list: %v", err)
	}

	return containers, nil

}

// Swarm are the informations required to connect to the swarm cluster.
type Swarm struct {
	Host string
	Port string

	client *docker.Client
}

// DockerHost returns the host to use to communicate with the swarm cluster.
func (s *Swarm) DockerHost() string {
	return fmt.Sprintf("tcp://%s:%s", s.Host, s.Port)
}

// Client returns a instance of the docker client.
func (s *Swarm) Client() (*docker.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	client, err := docker.NewClientWithOpts(
		docker.WithHost(s.DockerHost()),
		docker.WithVersion("1.39"),
	)

	if err != nil {
		return nil, err
	}

	s.client = client

	return client, nil
}

// Docker represents the informations required ot connect to the host daemon.
type Docker struct {
	Host string

	client *docker.Client
}

// Client returns a docker client, configured to communicate with the host dameon.
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
