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
	pushCmd = &cobra.Command{
		Use:   "push",
		Short: "Push an image to the cluster.",
		Run:   runPush,
	}

	filePath string
	jobs     int
)

func init() {
	rootCmd.AddCommand(pushCmd)

	pushCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to an image archive.")
	pushCmd.Flags().IntVarP(&jobs, "jobs", "j", 1, "How many pushes in parallel (0 means auto).")
}

func runPush(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx, cancel = internal.WithSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disgo.StartStep("Connecting to the docker daemon")

	client, err := docker.NewClientWithOpts(internal.DefaultDockerOpts...)
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

	if filePath != "" {
		pushFile(ctx, client, clusterName, filePath)
		return
	}

	disgo.StartStepf("Pushing images %q to cluster %q", args, clusterName)

	if err = sind.PushImageRefs(ctx, client, clusterInfo.Name, jobs, args); err != nil {
		fail(disgo.FailStepf("Unable to push images %q to %q: %v", args, clusterName, err))
	}

	disgo.EndStep()
	disgo.Infof("%s Successfully pushed images %q to cluster %q\n", style.Success(style.SymbolCheck), args, clusterName)
}

func pushFile(ctx context.Context, client *docker.Client, clusterName string, filePath string) {
	disgo.StartStepf("Pushing image archive at %q to cluster %q", filePath, clusterName)

	file, err := os.Open(filePath)
	if err != nil {
		fail(disgo.FailStepf("Unable to open file %q: %v", filePath, err))
	}
	defer file.Close()

	if err = sind.PushImageFile(ctx, client, clusterName, jobs, file); err != nil {
		fail(disgo.FailStepf("Unable to push image archive %q to %q: %v", filePath, clusterName, err))
	}

	disgo.EndStep()
	disgo.Infof("%s Successfully pushed images archive %q to cluster %q\n", style.Success(style.SymbolCheck), filePath, clusterName)
}
