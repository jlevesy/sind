package internal

import (
	"context"

	"github.com/docker/docker/api/types"
)

// ContainerListerMock is a stub for the ContainerListMethod.
type ContainerListerMock func(context.Context, types.ContainerListOptions) ([]types.Container, error)

// ContainerList returns the result of the mock.
func (c ContainerListerMock) ContainerList(ctx context.Context, opts types.ContainerListOptions) ([]types.Container, error) {
	return c(ctx, opts)
}
