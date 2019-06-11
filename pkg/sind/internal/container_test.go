package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListContainers(t *testing.T) {
	ctx := context.Background()
	var sentOpts types.ContainerListOptions
	clusterName := "supercluster"
	containers := []types.Container{
		{ID: "foo"},
		{ID: "bar"},
		{ID: "biz"},
	}

	mock := ContainerListerMock(func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
		sentOpts = opts

		return containers, nil
	})

	result, err := ListContainers(ctx, mock, clusterName)

	require.NoError(t, err)
	assert.Equal(t, containers, result)
	assert.True(t, sentOpts.Filters.MatchKVList("label", map[string]string{ClusterNameLabel: clusterName}))
}

func TestListContainersFailsOnListError(t *testing.T) {
	ctx := context.Background()
	mock := ContainerListerMock(func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
		return nil, errors.New("nope")
	})

	_, err := ListContainers(ctx, mock, "badCluster")
	assert.Error(t, err)
}

func TestPrimaryContainer(t *testing.T) {
	testCases := []struct {
		desc           string
		containers     []types.Container
		expectedResult *types.Container
		expectedError  error
		listError      error
	}{
		{
			desc:          "No containers found",
			containers:    []types.Container{},
			expectedError: errors.New("primary container for cluster \"blah\" not found"),
		},
		{
			desc:          "List error",
			listError:     errors.New("nope nope nope"),
			expectedError: errors.New("unable to list containers: nope nope nope"),
		},
		{
			desc:          "multiple containers found",
			containers:    []types.Container{types.Container{}, types.Container{}},
			expectedError: errors.New("primary container for cluster \"blah\" is not unique"),
		},
		{
			desc:           "primary container found",
			containers:     []types.Container{types.Container{ID: "123456789"}},
			expectedResult: &types.Container{ID: "123456789"},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var sentOpts types.ContainerListOptions
			clusterName := "blah"
			mock := ContainerListerMock(func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
				sentOpts = opts
				return test.containers, test.listError
			})

			result, err := PrimaryContainer(ctx, mock, clusterName)

			assert.True(
				t,
				sentOpts.Filters.MatchKVList(
					"label",
					map[string]string{
						ClusterNameLabel: clusterName,
						NodeRoleLabel:    "primary",
					},
				),
			)

			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			}

			if test.expectedResult != nil {
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

type containerStopperMock func(context.Context, string, *time.Duration) error

func (c containerStopperMock) ContainerStop(ctx context.Context, cID string, timeout *time.Duration) error {
	return c(ctx, cID, timeout)
}

func TestStopContainers(t *testing.T) {
	testCases := []struct {
		desc          string
		containers    []types.Container
		stopError     error
		expectedError error
	}{
		{
			desc: "failed to stop container",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
			stopError:     errors.New("still nope nope nope"),
			expectedError: errors.New("failed to stop at least one container: still nope nope nope"),
		},
		{
			desc:       "empty containers list",
			containers: []types.Container{},
		},
		{
			desc: "stops successfully",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			containerStopped := make(chan string, len(test.containers))
			mock := containerStopperMock(func(ctx context.Context, cID string, timeout *time.Duration) error {
				containerStopped <- cID
				return test.stopError
			})

			err := StopContainers(ctx, mock, test.containers)

			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			}

			close(containerStopped)

			var stoppedCIDs []string
			for cID := range containerStopped {
				stoppedCIDs = append(stoppedCIDs, cID)
			}

			assert.Len(t, stoppedCIDs, len(test.containers))
			for _, container := range test.containers {
				assert.Contains(t, stoppedCIDs, container.ID)
			}
		})
	}
}

type containerRemoverMock func(context.Context, string, types.ContainerRemoveOptions) error

func (c containerRemoverMock) ContainerRemove(ctx context.Context, cID string, opts types.ContainerRemoveOptions) error {
	return c(ctx, cID, opts)
}

func TestRemoveContainers(t *testing.T) {
	testCases := []struct {
		desc          string
		containers    []types.Container
		removeError   error
		expectedError error
	}{
		{
			desc: "failed to remove a container",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
			removeError:   errors.New("still nope nope nope"),
			expectedError: errors.New("failed to remove at least one container: still nope nope nope"),
		},
		{
			desc:       "empty containers list",
			containers: []types.Container{},
		},
		{
			desc: "removes successfully",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			sentOpts := make(chan types.ContainerRemoveOptions, len(test.containers))
			containerStopped := make(chan string, len(test.containers))
			mock := containerRemoverMock(func(ctx context.Context, cID string, opts types.ContainerRemoveOptions) error {
				sentOpts <- opts
				containerStopped <- cID
				return test.removeError
			})

			err := RemoveContainers(ctx, mock, test.containers)

			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			}

			close(containerStopped)
			close(sentOpts)

			for opt := range sentOpts {
				assert.True(t, opt.Force)
				assert.True(t, opt.RemoveVolumes)
			}

			var removedCIDs []string
			for cID := range containerStopped {
				removedCIDs = append(removedCIDs, cID)
			}

			assert.Len(t, removedCIDs, len(test.containers))
			for _, container := range test.containers {
				assert.Contains(t, removedCIDs, container.ID)
			}
		})
	}
}

type containerStarterMock func(context.Context, string, types.ContainerStartOptions) error

func (c containerStarterMock) ContainerStart(ctx context.Context, cID string, opts types.ContainerStartOptions) error {
	return c(ctx, cID, opts)
}

func TestStartContainers(t *testing.T) {
	testCases := []struct {
		desc          string
		containers    []types.Container
		startError    error
		expectedError error
	}{
		{
			desc: "failed to start container",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
			startError:    errors.New("still nope nope nope"),
			expectedError: errors.New("failed to start at least one container: still nope nope nope"),
		},
		{
			desc:       "empty containers list",
			containers: []types.Container{},
		},
		{
			desc: "starts successfully",
			containers: []types.Container{
				{ID: "aaaaa"},
				{ID: "bbbbb"},
				{ID: "ccccc"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			containerStarted := make(chan string, len(test.containers))
			mock := containerStarterMock(func(ctx context.Context, cID string, opts types.ContainerStartOptions) error {
				containerStarted <- cID
				return test.startError
			})

			err := StartContainers(ctx, mock, test.containers)

			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			}

			close(containerStarted)

			var startedCIDs []string
			for cID := range containerStarted {
				startedCIDs = append(startedCIDs, cID)
			}

			assert.Len(t, startedCIDs, len(test.containers))
			for _, container := range test.containers {
				assert.Contains(t, startedCIDs, container.ID)
			}
		})
	}
}
