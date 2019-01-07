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
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Printf("Creating a new cluster with %d managers and %d workers\n", managers, workers)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clusterParams := sind.CreateClusterParams{
		Managers:    managers,
		Workers:     workers,
		NetworkName: networkName,
	}
	cluster, err := sind.CreateCluster(ctx, clusterParams)
	if err != nil {
		fmt.Printf("Unable to setup a swarm cluster: %v\n", err)
		os.Exit(1)
	}

	defer cluster.Delete(ctx)
}
