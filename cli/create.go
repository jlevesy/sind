package cli

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"

	"github.com/jlevesy/go-sind/sind"
)

var (
	managers     int
	workers      int
	networkName  string
	portsMapping []string

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new swarm cluster.",
		Run:   runCreate,
	}
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().IntVarP(&managers, "managers", "m", 1, "Amount of managers in the created cluster.")
	createCmd.Flags().IntVarP(&workers, "workers", "w", 0, "Amount of workers in the created cluster.")
	createCmd.Flags().StringVarP(&networkName, "network_name", "n", "sind_default", "Name of the network to create.")
	createCmd.Flags().StringSliceVarP(&portsMapping, "ports", "p", []string{}, "Ingress network port binding.")
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Printf("Creating a new cluster %q with %d managers and %d, workers...\n", clusterName, managers, workers)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	store, err := NewStore()
	if err != nil {
		fmt.Printf("unable to create store: %v\n", err)
		os.Exit(1)
	}

	if err := store.ValidateName(clusterName); err != nil {
		fmt.Printf("invalid cluster name: %v\n", err)
		os.Exit(1)
	}

	clusterParams := sind.CreateClusterParams{
		Managers:     managers,
		Workers:      workers,
		NetworkName:  networkName,
		ClusterName:  clusterName,
		PortBindings: portsMapping,
	}

	cluster, err := sind.CreateCluster(ctx, clusterParams)
	if err != nil {
		fmt.Printf("unable to setup a swarm cluster: %v\n", err)
		os.Exit(1)
	}

	if err = store.Save(*cluster); err != nil {
		fmt.Printf("unable to save cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cluster %s successfuly created !\n", clusterName)
}
