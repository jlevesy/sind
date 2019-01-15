package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	imageRef = ""

	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an image to the swarm cluster.",
		Run:   runDeploy,
	}
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&imageRef, "image", "i", "", "Name of the image to deploy.")
}

func runDeploy(cmd *cobra.Command, args []string) {
	fmt.Printf("Deploying image %s in cluster %s\n", imageRef, clusterName)
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

	if err = cluster.DeployImage(ctx, imageRef); err != nil {
		fail("unable to deploy %s to the cluster: %v", imageRef, err)
	}

	fmt.Printf("%s successfuly deployed to %s!\n", imageRef, clusterName)
}
