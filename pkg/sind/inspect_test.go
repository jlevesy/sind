package sind

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jlevesy/sind/pkg/sind/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectCluster(t *testing.T) {
	testCases := []struct {
		desc                 string
		discoveredContainers []types.Container
		expectedStatus       *ClusterStatus
	}{
		{
			desc:                 "with no discovered containers",
			discoveredContainers: []types.Container{},
			expectedStatus:       nil,
		},
		{
			desc: "with containers",

			discoveredContainers: []types.Container{
				{
					State: "running",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRolePrimary,
					},
				},
				{
					State: "running",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRoleManager,
					},
				},
				{
					State: "stopped",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRoleManager,
					},
				},
				{
					State: "running",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRoleWorker,
					},
				},
				{
					State: "running",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRoleWorker,
					},
				},
				{
					State: "stopped",
					Labels: map[string]string{
						internal.NodeRoleLabel: internal.NodeRoleWorker,
					},
				},
			},
			expectedStatus: &ClusterStatus{
				Managers:        3,
				ManagersRunning: 2,
				Workers:         3,
				WorkersRunning:  2,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			clusterName := "foo"
			client := internal.ContainerListerMock(
				func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
					return test.discoveredContainers, nil
				},
			)

			res, err := InspectCluster(ctx, client, clusterName)
			require.NoError(t, err)

			if test.expectedStatus == nil {
				assert.Nil(t, res)
				return
			}

			assert.Equal(t, test.expectedStatus.Managers, res.Managers)
			assert.Equal(t, test.expectedStatus.ManagersRunning, res.ManagersRunning)
			assert.Equal(t, test.expectedStatus.Workers, res.Workers)
			assert.Equal(t, test.expectedStatus.WorkersRunning, res.WorkersRunning)
			assert.Equal(t, test.discoveredContainers, res.Nodes)
		})
	}
}
