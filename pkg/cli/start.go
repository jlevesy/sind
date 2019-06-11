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
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a sind cluster.",
		Run:   runStart,
	}
)

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx, cancel = internal.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disgo.StartStep("Connecting to the docker daemon")
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		_ = disgo.FailStepf("Unable to connect to the docker daemon: %v", err)
		os.Exit(1)
	}

	disgo.StartStepf("Checking if a cluster named %q already exists", clusterName)
	clusterInfo, err := sind.InspectCluster(ctx, client, clusterName)
	if err != nil {
		_ = disgo.FailStepf("Unable to check if the cluster already exists: %v", err)
		os.Exit(1)
	}

	if clusterInfo == nil {
		_ = disgo.FailStepf("Cluster %q does not exists", clusterName)
		os.Exit(1)
	}

	disgo.StartStepf("Starting cluster %q", clusterName)
	if err = sind.StartCluster(ctx, client, clusterInfo.Name); err != nil {
		_ = disgo.FailStepf("Unable to start cluster %q: %v", clusterInfo.Name, err)
		os.Exit(1)
	}

	disgo.EndStep()
	disgo.Infof("%s Cluster %q successfuly started\n", style.Success(style.SymbolCheck), clusterName)
}
