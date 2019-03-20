package cli

import (
	"context"
	"fmt"

	"github.com/jlevesy/sind/pkg/store"
	"github.com/spf13/cobra"
)

var (
	pushCmd = &cobra.Command{
		Use:   "push",
		Short: "Push an image to the swarm cluster.",
		Run:   runPush,
	}
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) {
	fmt.Printf("Pushing images %v to the cluster %s...\n", args, clusterName)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	st, err := store.New()
	if err != nil {
		fail("unable to create store: %v\n", err)
	}

	cluster, err := st.Load(clusterName)
	if err != nil {
		fail("unable to load cluster: %v\n", err)
	}

	if err = cluster.PushImage(ctx, args); err != nil {
		fail("unable to push %v to the cluster: %v", args, err)
	}

	fmt.Printf("Images %v successfuly pushed to %s !\n", args, clusterName)
}
