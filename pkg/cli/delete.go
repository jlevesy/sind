package cli

import (
	"context"
	"os"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a swarm cluster.",
		Run:   runDelete,
	}
)

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	disgo.StartStep("Connecting to the docker daemon")

	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		disgo.FailStepf("Unable to connect to the docker daemon: %v", err)
		os.Exit(1)
	}

	disgo.StartStepf("Checking if a cluster named %q exists", clusterName)
	clusterInfo, err := sind.InspectCluster(ctx, client, clusterName)
	if err != nil {
		disgo.FailStepf("Unable to check if the cluster exists: %v", err)
		os.Exit(1)
	}

	if clusterInfo == nil {
		disgo.FailStepf("Cluster %q does not exist, or is already deleted\n", clusterName)
		os.Exit(1)
	}

	disgo.StartStepf("Deleting cluster %q", clusterName)
	if err = sind.DeleteCluster(ctx, client, clusterName); err != nil {
		disgo.FailStepf("Unable to delete the cluster %q: %v", clusterName, err)
		os.Exit(1)
	}

	disgo.EndStep()
	disgo.Infof("%s Cluster %s successfuly deleted !\n", style.Success(style.SymbolCheck), clusterName)
}
