package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	envCmd = &cobra.Command{
		Use:   "env",
		Short: "Sets up docker env variables.",
		Run:   runEnv,
	}
)

func init() {
	rootCmd.AddCommand(envCmd)
}

func runEnv(cmd *cobra.Command, args []string) {
	store, err := NewStore()
	if err != nil {
		fail("unable to create store: %v\n", err)
	}

	cluster, err := store.Load(clusterName)
	if err != nil {
		fail("unable to load cluster: %v\n", err)
	}

	fmt.Printf("export DOCKER_HOST=%s", cluster.Cluster.DockerHost())
}
