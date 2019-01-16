package cli

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"

	"github.com/jlevesy/go-sind/sind"
)

var (
	managers     = 0
	workers      = 0
	networkName  = ""
	portsMapping = []string{}

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
	fmt.Printf("Creating a new cluster with %d managers and %d, workers\n", managers, workers)

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
		PortBindings: preparePorts(portsMapping),
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

	fmt.Printf("cluster %s successfuly created !\n", clusterName)
}

func preparePorts(raw []string) map[string]string {
	ports := map[string]string{}

	for _, pb := range raw {
		split := strings.Split(pb, ":")
		if len(split) != 2 {
			continue
		}
		ports[split[0]] = split[1]
	}

	return ports
}
