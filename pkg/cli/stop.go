package cli

import (
	"context"
	"fmt"

	"github.com/jlevesy/sind/pkg/store"
	"github.com/spf13/cobra"
)

var (
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop a sind cluster.",
		Run:   runStop,
	}
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) {
	fmt.Printf("Stopping cluster %s \n", clusterName)
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

	if err = cluster.Stop(ctx); err != nil {
		fail("unable to stop cluster: %v", err)
	}

	fmt.Printf("Cluster %s stopped\n", clusterName)
}
