package internal

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type containerStarterMock struct {
	containerCreate func(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, string) (container.ContainerCreateCreatedBody, error)
	containerStart  func(context.Context, string, types.ContainerStartOptions) error
}

func (s containerStarterMock) ContainerCreate(ctx context.Context, ccfg *container.Config, hcfg *container.HostConfig, ncfg *network.NetworkingConfig, cName string) (container.ContainerCreateCreatedBody, error) {
	return s.containerCreate(ctx, ccfg, hcfg, ncfg, cName)
}
func (s containerStarterMock) ContainerStart(ctx context.Context, cID string, opts types.ContainerStartOptions) error {
	return s.containerStart(ctx, cID, opts)
}

type fakeContainer struct {
	name string

	cConfig *container.Config
	hConfig *container.HostConfig
	nConfig *network.NetworkingConfig
}

func TestCreateNodesFailsWhenPortBindingsIsInvalid(t *testing.T) {
	ctx := context.Background()
	cfg := NodesConfig{
		PortBindings: []string{"notaport"},
	}
	mock := containerStarterMock{}

	_, err := CreateNodes(ctx, mock, cfg)

	assert.Error(t, err)
}

func TestCreateNodes(t *testing.T) {
	ctx := context.Background()
	cfg := NodesConfig{
		ClusterName:  "TestCluster",
		ImageRef:     "foo",
		NetworkID:    "ababababab",
		NetworkName:  "bar",
		PortBindings: []string{"8080:8080"},
		Managers:     3,
		Workers:      3,
	}

	containerCreated := make(chan *fakeContainer, cfg.Managers+cfg.Workers)
	containerRun := make(chan string, cfg.Managers+cfg.Workers)

	mock := containerStarterMock{
		containerCreate: func(ctx context.Context, cConfig *container.Config, hConfig *container.HostConfig, nConfig *network.NetworkingConfig, cName string) (container.ContainerCreateCreatedBody, error) {
			containerCreated <- &fakeContainer{
				name:    cName,
				cConfig: cConfig,
				hConfig: hConfig,
				nConfig: nConfig,
			}

			return container.ContainerCreateCreatedBody{ID: cName}, nil
		},
		containerStart: func(ctx context.Context, cID string, opts types.ContainerStartOptions) error {
			containerRun <- cID
			return nil
		},
	}

	cIDs, err := CreateNodes(ctx, mock, cfg)
	require.NoError(t, err)

	close(containerCreated)
	close(containerRun)

	t.Log(cIDs)

	var (
		createdContainers []*fakeContainer
		ranContainers     []string
	)

	for c := range containerCreated {
		createdContainers = append(createdContainers, c)
	}

	for r := range containerRun {
		ranContainers = append(ranContainers, r)
	}

	assert.Equal(t, cfg.Managers+cfg.Workers, uint16(len(createdContainers)))
	assert.Equal(t, cfg.Managers+cfg.Workers, uint16(len(ranContainers)))

	sort.Slice(createdContainers, func(i, j int) bool {
		return createdContainers[i].cConfig.Hostname < createdContainers[j].cConfig.Hostname
	})

	primary := createdContainers[0]
	managers := createdContainers[1:3]
	workers := createdContainers[3:]

	// primary node container
	assert.Equal(t, "sind-TestCluster-manager-0", primary.name)
	assert.Equal(
		t,
		&container.Config{
			Hostname:     "sind-TestCluster-manager-0",
			Image:        cfg.ImageRef,
			ExposedPorts: nat.PortSet(map[nat.Port]struct{}{nat.Port("8080/tcp"): struct{}{}}),
			Labels: map[string]string{
				"com.sind.cluster.name": "TestCluster",
				"com.sind.cluster.role": "primary",
			},
		},
		primary.cConfig,
	)

	assert.Equal(
		t,
		&container.HostConfig{
			Privileged:      true,
			PublishAllPorts: true,
			PortBindings: map[nat.Port][]nat.PortBinding{
				nat.Port("8080/tcp"): []nat.PortBinding{
					{HostPort: "8080"},
				},
			},
		},
		primary.hConfig,
	)

	assert.Equal(
		t,
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cfg.NetworkName: {
					NetworkID: cfg.NetworkID,
					IPAddress: "10.0.117.1",
				},
			},
		},
		primary.nConfig,
	)

	// managers
	for i, c := range managers {
		expectedContainerName := fmt.Sprintf("sind-TestCluster-manager-%d", i+1)
		assert.Equal(t, expectedContainerName, c.name)
		assert.Equal(
			t,
			&container.Config{
				Hostname: expectedContainerName,
				Image:    cfg.ImageRef,
				Labels: map[string]string{
					"com.sind.cluster.name": "TestCluster",
					"com.sind.cluster.role": "manager",
				},
			},
			c.cConfig,
		)

		assert.Equal(
			t,
			&container.HostConfig{Privileged: true},
			c.hConfig,
		)

		assert.Equal(
			t,
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					cfg.NetworkName: {
						NetworkID: cfg.NetworkID,
						IPAddress: fmt.Sprintf("10.0.117.%d", i+2),
					},
				},
			},
			c.nConfig,
		)
	}

	// workers
	for i, c := range workers {
		expectedContainerName := fmt.Sprintf("sind-TestCluster-worker-%d", i)
		assert.Equal(t, expectedContainerName, c.name)
		assert.Equal(
			t,
			&container.Config{
				Hostname: expectedContainerName,
				Image:    cfg.ImageRef,
				Labels: map[string]string{
					"com.sind.cluster.name": "TestCluster",
					"com.sind.cluster.role": "worker",
				},
			},
			c.cConfig,
		)

		assert.Equal(
			t,
			&container.HostConfig{Privileged: true},
			c.hConfig,
		)

		assert.Equal(
			t,
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					cfg.NetworkName: {
						NetworkID: cfg.NetworkID,
						IPAddress: fmt.Sprintf("10.0.117.%d", i+2+len(managers)),
					},
				},
			},
			c.nConfig,
		)
	}

	// assert about the ran containers.
	sort.Strings(ranContainers)
	assert.Equal(
		t,
		[]string{
			"sind-TestCluster-manager-0",
			"sind-TestCluster-manager-1",
			"sind-TestCluster-manager-2",
			"sind-TestCluster-worker-0",
			"sind-TestCluster-worker-1",
			"sind-TestCluster-worker-2",
		},
		ranContainers,
	)

	sort.Strings(cIDs.Managers)
	sort.Strings(cIDs.Workers)
	// assert about retunred Ids
	assert.Equal(t, cIDs, &NodeIDs{
		Primary: "sind-TestCluster-manager-0",
		Managers: []string{
			"sind-TestCluster-manager-1",
			"sind-TestCluster-manager-2",
		},
		Workers: []string{
			"sind-TestCluster-worker-0",
			"sind-TestCluster-worker-1",
			"sind-TestCluster-worker-2",
		},
	})
}
