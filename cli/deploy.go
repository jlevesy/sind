package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an image to the swarm cluster.",
		Run:   runDeploy,
	}
)

func init() {
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	fmt.Printf("Deploying images %v in cluster %s\n", args, clusterName)
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

	if err = cluster.DeployImage(ctx, args); err != nil {
		fail("unable to deploy %v to the cluster: %v", args, err)
	}

	fmt.Printf("%v successfuly deployed to %s!\n", args, clusterName)
}
