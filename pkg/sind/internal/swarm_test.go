package internal

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwarmPort(t *testing.T) {
	testCases := []struct {
		desc           string
		container      types.Container
		expectedResult uint16
		expectedError  error
	}{{
		desc: "not exposing docker daemon port",
		container: types.Container{
			Ports: []types.Port{
				{PrivatePort: 1235},
				{PrivatePort: 1236},
				{PrivatePort: 1238},
				{PrivatePort: 1230},
			},
		},
		expectedError: errors.New("container does not export port 2375"),
	},
		{
			desc: "exposing docker daemon port",
			container: types.Container{
				Ports: []types.Port{
					{PrivatePort: 1235},
					{PrivatePort: 1236},
					{PrivatePort: dockerDaemonPort, PublicPort: 30493},
					{PrivatePort: 1230},
				},
			},
			expectedResult: 30493,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			res, err := SwarmPort(test.container)
			assert.Equal(t, test.expectedError, err)
			assert.Equal(t, test.expectedResult, res)
		})
	}
}

type hosterMock func() string

func (h hosterMock) DaemonHost() string {
	return h()
}

func TestSwarmHost(t *testing.T) {
	testCases := []struct {
		desc         string
		daemonHost   string
		expectsError bool
		expectedHost string
	}{
		{
			desc:         "with an unix host",
			daemonHost:   "unix:///foo/bar",
			expectedHost: "localhost",
		},
		{
			desc:         "with a non unix host",
			daemonHost:   "tcp://foobarbuz",
			expectedHost: "foobarbuz",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			hoster := hosterMock(func() string { return test.daemonHost })
			res, err := SwarmHost(hoster)
			if test.expectsError {
				assert.Error(t, err)
				t.SkipNow()
			}

			assert.Equal(t, test.expectedHost, res)
		})
	}
}

func TestFormCluster(t *testing.T) {
	ctx := context.Background()
	params := ClusterParams{
		IDs: NodeIDs{
			Primary:  "a",
			Managers: []string{"b", "c"},
			Workers:  []string{"d", "e", "f"},
		},

		PrimaryNodeIP:    "10.0.0.1",
		ManagerJoinToken: "zz",
		WorkerJoinToken:  "hh",
	}

	type execCreation struct {
		cID string
		Cmd []string
	}

	execCreated := make(chan execCreation, len(params.IDs.Managers)+len(params.IDs.Workers))
	execStarted := make(chan string, len(params.IDs.Managers)+len(params.IDs.Workers))

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

	require.NoError(t, FormCluster(ctx, &client, params))

	close(execCreated)
	close(execStarted)

	var (
		createdExecs     []execCreation
		createdExecscIDs []string
		startedExecs     []string
	)

	for e := range execCreated {
		createdExecs = append(createdExecs, e)
		createdExecscIDs = append(createdExecscIDs, e.cID)
	}

	for e := range execStarted {
		startedExecs = append(startedExecs, e)
	}

	cIDs := append(params.IDs.Managers, params.IDs.Workers...)

	sort.Strings(cIDs)
	sort.Slice(createdExecs, func(i, j int) bool { return createdExecs[i].cID < createdExecs[j].cID })
	sort.Strings(createdExecscIDs)
	sort.Strings(startedExecs)

	// Assert that all cIDs have execs created.
	assert.Equal(t, cIDs, createdExecscIDs)

	// Assert that all cIDs have expected commands
	for _, e := range createdExecs {
		switch e.cID {
		case "b", "c":
			assert.Equal(
				t,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					params.ManagerJoinToken,
					params.PrimaryNodeIP + ":" + strconv.Itoa(swarmGossipPort),
				},
				e.Cmd,
			)

		case "d", "e", "f":
			assert.Equal(
				t,
				[]string{
					"docker",
					"swarm",
					"join",
					"--token",
					params.WorkerJoinToken,
					params.PrimaryNodeIP + ":" + strconv.Itoa(swarmGossipPort),
				},
				e.Cmd,
			)
		}
	}

	// Assert that all the created execs are executed.
	assert.Equal(t, cIDs, startedExecs)
}
