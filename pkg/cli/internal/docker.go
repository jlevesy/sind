package internal

import (
	docker "github.com/docker/docker/client"
)

// DefaultDockerOpts are the default docker options to use when interacting with the local docker daemon.
var DefaultDockerOpts = []docker.Opt{
	docker.FromEnv,
	docker.WithAPIVersionNegotiation(),
}
