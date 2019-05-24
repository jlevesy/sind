package cli

import (
	"context"
	"fmt"

	"github.com/jlevesy/sind/pkg/store"
	"github.com/spf13/cobra"
)

var (
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a sind cluster.",
		Run:   runStart,
	}
)

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Printf("Starting cluster %s \n", clusterName)
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

	if err = cluster.Start(ctx); err != nil {
		fail("unable to start cluster: %v", err)
	}

	fmt.Printf("Started cluster %s\n", clusterName)
}
