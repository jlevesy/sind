package cli

import (
	"context"
	"fmt"
	"os"

	docker "github.com/docker/docker/client"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/spf13/cobra"
)

var (
	managers      uint16
	workers       uint16
	networkName   string
	networkSubnet string
	portsMapping  []string
	nodeImageName string
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
	createCmd.Flags().StringVarP(&networkSubnet, "network-subnet", "s", "", "Subnet in CIDR format that represents a network segment.")
	createCmd.Flags().StringSliceVarP(&portsMapping, "ports", "p", []string{}, "Ingress network port binding.")
	createCmd.Flags().StringVarP(&nodeImageName, "image", "i", sind.DefaultNodeImageName, "Name of the image to use for the nodes.")
	createCmd.Flags().BoolVarP(&pull, "pull", "", false, "Pull node image before creating the cluster.")
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Printf("Creating a new cluster %q with %d managers and %d, workers...\n", clusterName, managers, workers)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithVersion("1.39"))
	if err != nil {
		fmt.Printf("unable to create docker client: %v", err)
		os.Exit(1)
	}

	// TODO check if a cluster exists and fail appropriately.
	// clusterInfo, err := sind.InspectCluster(ctx, client, clusterName)
	// if err != nil {
	//	return err
	// }

	clusterConfig := sind.ClusterConfiguration{
		Managers:      managers,
		Workers:       workers,
		NetworkName:   networkName,
		NetworkSubnet: networkSubnet,
		ClusterName:   clusterName,
		PortBindings:  portsMapping,
		ImageName:     nodeImageName,
		PullImage:     pull,
	}

	if err := sind.CreateCluster(ctx, client, clusterConfig); err != nil {
		fmt.Printf("unable to create a swarm cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cluster %s successfuly created !\n", clusterName)
}
