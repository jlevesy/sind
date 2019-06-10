package internal

import (
	"context"
	"os"
	"os/signal"
)

// WithSignal returns a context cancelled if the process receives one of the following signal.
func WithSignal(parent context.Context, signals ...os.Signal) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)

	signalReceived := make(chan os.Signal, 1)
	signal.Notify(signalReceived, signals...)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-signalReceived:
				cancel()
				return
			}
		}
	}()

	return ctx, cancel
}
