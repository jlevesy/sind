package cli

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"time"

	"github.com/jlevesy/go-sind/sind"
)

var (
	managers    = 0
	workers     = 0
	networkName = ""
	clusterName = ""

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new swarm cluster.",
		Run:   runCreate,
	}
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&clusterName, "cluster", "c", "sind_default", "Cluster name.")
	createCmd.Flags().IntVarP(&managers, "managers", "m", 1, "Amount of managers in the created cluster.")
	createCmd.Flags().IntVarP(&workers, "workers", "w", 0, "Amount of workers in the created cluster.")
	createCmd.Flags().StringVarP(&networkName, "network_name", "n", "sind_default", "Name of the network to create.")
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Printf("Creating a new cluster with %d managers and %d workers\n", managers, workers)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := NewStore()
	if err != nil {
		fmt.Printf("unable to create store: %v\n", err)
		os.Exit(1)
	}

	if err := store.ValidateName(clusterName); err != nil {
		fmt.Printf("invalid cluster name: %v", err)
		os.Exit(1)
	}

	clusterParams := sind.CreateClusterParams{
		Managers:    managers,
		Workers:     workers,
		NetworkName: networkName,
	}

	cluster, err := sind.CreateCluster(ctx, clusterParams)
	if err != nil {
		fmt.Printf("unable to setup a swarm cluster: %v\n", err)
		os.Exit(1)
	}

	if err = store.Save(clusterName, *cluster); err != nil {
		fmt.Printf("unable to save cluster: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cluster %s successfuly created !\n", clusterName)
}
