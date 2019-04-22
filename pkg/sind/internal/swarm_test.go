package internal

import (
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

func TestSwarmPort(t *testing.T) {
	testCases := []struct {
		desc           string
		container      types.Container
		expectedResult uint16
		expectedError  error
	}{
		{
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
