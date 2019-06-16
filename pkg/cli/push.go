package cli

import (
	"context"
	"syscall"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/cli/internal"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
	"github.com/ullaakut/disgo"
)

var (
	pushCmd = &cobra.Command{
		Use:   "push",
		Short: "Push an image to the cluster.",
		Run:   runPush,
	}
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx, cancel = internal.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disgo.StartStep("Connecting to the docker daemon")
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		fail(disgo.FailStepf("Unable to connect to the docker daemon: %v", err))
	}

	disgo.StartStepf("Checking if a cluster named %q already exists", clusterName)
	clusterInfo, err := sind.InspectCluster(ctx, client, clusterName)
	if err != nil {
		fail(disgo.FailStepf("Unable to check if the cluster already exists: %v", err))
	}

	if clusterInfo == nil {
		fail(disgo.FailStepf("Cluster %q does not exists", clusterName))
	}

	disgo.StartStepf("Pushing images %q to cluster %q", args, clusterName)
	if err = sind.PushImageRefs(ctx, client, clusterInfo.Name, args); err != nil {
		fail(disgo.FailStepf("Unable to push images %q to %q: %v", args, clusterName, err))
	}

	disgo.EndStep()
}
