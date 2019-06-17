package internal

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
)

type pinger interface {
	Ping(context.Context) (types.Ping, error)
}

// WaitDaemonReady waits until remote daemon is ready.
func WaitDaemonReady(ctx context.Context, client pinger) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := client.Ping(ctx)
			if err != nil {
				continue
			}

			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
