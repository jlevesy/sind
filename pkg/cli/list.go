package cli

import (
	"context"
	"os"
	"syscall"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/cli/internal"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

var (
	listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sind clusters.",
		Run:     runList,
	}
)

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx, cancel = internal.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disgo.StartStep("Connecting to the docker daemon")

	client, err := docker.NewClientWithOpts(internal.DefaultDockerOpts...)
	if err != nil {
		fail(disgo.FailStepf("Unable to connect to the docker daemon: %v", err))
	}

	disgo.StartStep("Listing clusters")

	clusters, err := sind.ListClusters(ctx, client)
	if err != nil {
		fail(disgo.FailStepf("Unable to list clusters: %v", err))
	}

	disgo.EndStep()
	disgo.Infof("%s Found %d cluster(s)\n", style.Success(style.SymbolCheck), len(clusters))

	if len(clusters) == 0 {
		return
	}

	internal.RenderClusterList(os.Stdout, clusters)
}
