package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Printf("Deleting cluster %s\n", clusterName)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	store, err := NewStore()
	if err != nil {
		fail("unable to create store: %v\n", err)
	}

	cluster, err := store.Load(clusterName)
	if err != nil {
		fail("unable to load cluster: %v\n", err)
	}

	if err = cluster.Delete(ctx); err != nil {
		fail("unable to tear down cluster: %v", err)
	}

	if err = store.Delete(clusterName); err != nil {
		fail("unable to delete cluster from storage: %v", err)
	}

	fmt.Printf("Cluster %s successfuly deleted !\n", clusterName)
}
