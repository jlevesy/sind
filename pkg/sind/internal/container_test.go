package internal

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPrimaryContainers(t *testing.T) {
	testCases := []struct {
		desc         string
		listError    error
		containers   []types.Container
		expectsError bool
	}{
		{
			desc: "without error",
			containers: []types.Container{
				{ID: "Foo"},
			},
			listError:    nil,
			expectsError: false,
		},
		{
			desc: "with error",
			containers: []types.Container{
				{ID: "Foo"},
			},
			listError:    errors.New("foo"),
			expectsError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var sentOpts types.ContainerListOptions
			client := ContainerListerMock(func(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {

				sentOpts = opts
				return test.containers, test.listError
			})
			clusterName := "test"

			res, err := ListPrimaryContainers(ctx, client)
			if test.expectsError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.containers, res)
			assert.True(t, sentOpts.Filters.ExactMatch(ClusterNameLabel, clusterName))

		})
	}
}

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

type containerContentCopierMock func(context.Context, string, string, io.Reader, types.CopyToContainerOptions) error

func (c containerContentCopierMock) CopyToContainer(ctx context.Context, cID, path string, content io.Reader, opts types.CopyToContainerOptions) error {
	return c(ctx, cID, path, content, opts)
}

type sentContent struct {
	cID     string
	path    string
	content []byte
}

func TestCopyToContainers(t *testing.T) {
	ctx := context.Background()

	initialContent := []byte("content")
	fileContent, err := ioutil.TempFile(os.TempDir(), "test_sind_copy_container")
	require.NoError(t, err)

	defer os.Remove(fileContent.Name())
	defer fileContent.Close()

	_, err = fileContent.Write(initialContent)
	require.NoError(t, err)

	_, err = fileContent.Seek(0, 0)
	require.NoError(t, err)

	destPath := "/toto"
	containers := []types.Container{
		{ID: "AAA"},
		{ID: "BBB"},
		{ID: "CCC"},
	}

	contentSent := make(chan sentContent, len(containers))

	client := containerContentCopierMock(func(ctx context.Context, cID, path string, file io.Reader, opts types.CopyToContainerOptions) error {
		contentBytes, err := ioutil.ReadAll(file)
		require.NoError(t, err)

		contentSent <- sentContent{
			cID:     cID,
			path:    path,
			content: contentBytes,
		}
		return nil
	})

	require.NoError(t, CopyToContainers(ctx, client, containers, fileContent.Name(), destPath))

	close(contentSent)

	var sentContent []sentContent
	for content := range contentSent {
		sentContent = append(sentContent, content)
	}

	sort.Slice(sentContent, func(i, j int) bool { return sentContent[i].cID < sentContent[j].cID })

	assert.Len(t, sentContent, len(containers))
	for index, content := range sentContent {
		assert.Equal(t, containers[index].ID, content.cID)
		assert.Equal(t, destPath, content.path)
		assert.Equal(t, initialContent, content.content)
	}
}

type executorMock struct {
	containerExecCreate func(context.Context, string, types.ExecConfig) (types.IDResponse, error)
	containerExecStart  func(context.Context, string, types.ExecStartCheck) error
}

func (e *executorMock) ContainerExecCreate(ctx context.Context, cID string, opts types.ExecConfig) (types.IDResponse, error) {
	return e.containerExecCreate(ctx, cID, opts)
}

func (e *executorMock) ContainerExecStart(ctx context.Context, eID string, opts types.ExecStartCheck) error {
	return e.containerExecStart(ctx, eID, opts)
}

func TestExecContainers(t *testing.T) {
	ctx := context.Background()

	type execCreation struct {
		cID string
		Cmd []string
	}

	cmd := []string{"echo", "foo"}

	containers := []types.Container{
		{ID: "AAA"},
		{ID: "BBB"},
		{ID: "CCC"},
	}

	execCreated := make(chan execCreation, len(containers))
	execStarted := make(chan string, len(containers))

	client := executorMock{
		containerExecCreate: func(ctx context.Context, cID string, opts types.ExecConfig) (types.IDResponse, error) {
			assert.True(t, opts.AttachStdout)
			assert.True(t, opts.AttachStderr)
			execCreated <- execCreation{cID: cID, Cmd: opts.Cmd}
			return types.IDResponse{
				ID: cID,
			}, nil
		},
		containerExecStart: func(ctx context.Context, eID string, opts types.ExecStartCheck) error {
			execStarted <- eID
			return nil
		},
	}

	require.NoError(t, ExecContainers(ctx, &client, containers, cmd))

	close(execCreated)
	close(execStarted)

	var (
		createdExecs []execCreation
		startedExecs []string
	)

	for e := range execCreated {
		createdExecs = append(createdExecs, e)
	}

	for e := range execStarted {
		startedExecs = append(startedExecs, e)
	}

	sort.Slice(createdExecs, func(i, j int) bool { return createdExecs[i].cID < createdExecs[j].cID })
	sort.Strings(startedExecs)

	for index, container := range containers {
		assert.Equal(t, container.ID, createdExecs[index].cID)
		assert.Equal(t, container.ID, startedExecs[index])
		assert.Equal(t, cmd, createdExecs[index].Cmd)
	}
}
