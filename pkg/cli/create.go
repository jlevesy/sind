package cli

import (
	"context"
	"syscall"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/cli/internal"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

var (
	managers      uint16
	workers       uint16
	networkName   string
	portsMapping  []string
	nodeImageName string
	daemonArgs    []string
	pull          bool

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new swarm cluster.",
		Run:   runCreate,
	}
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().Uint16VarP(&managers, "managers", "m", 1, "Amount of managers in the created cluster.")
	createCmd.Flags().Uint16VarP(&workers, "workers", "w", 0, "Amount of workers in the created cluster.")
	createCmd.Flags().StringVarP(&networkName, "network-name", "n", "sind-default", "Name of the network to create.")
	createCmd.Flags().StringSliceVarP(&portsMapping, "ports", "p", []string{}, "Ingress network port binding.")
	createCmd.Flags().StringSliceVarP(&daemonArgs, "daemon-arg", "", []string{}, "Args to pass to nodes docker daemon")
	createCmd.Flags().StringVarP(&nodeImageName, "image", "i", sind.DefaultNodeImageName, "Name of the image to use for the nodes.")
	createCmd.Flags().BoolVarP(&pull, "pull", "", false, "Pull node image before creating the cluster.")
}

func runCreate(cmd *cobra.Command, args []string) {
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

	// If cluster info is not nil, then the cluster exist.
	if clusterInfo != nil {
		fail(disgo.FailStepf("Cluster %q already exists, run sind delete first to remove it.", clusterName))
	}

	disgo.StartStepf("Creating a new cluster %q with %d managers and %d workers", clusterName, managers, workers)

	clusterConfig := sind.ClusterConfiguration{
		Managers:     managers,
		Workers:      workers,
		NetworkName:  networkName,
		ClusterName:  clusterName,
		PortBindings: portsMapping,
		ImageName:    nodeImageName,
		PullImage:    pull,
		DaemonArgs:   daemonArgs,
	}

	if err := sind.CreateCluster(ctx, client, clusterConfig); err != nil {
		fail(disgo.FailStepf("Unable to create cluster %q: %v", clusterName, err))
	}

	disgo.EndStep()
	disgo.Infof("%s Cluster %q successfully created\n", style.Success(style.SymbolCheck), clusterName)
}
