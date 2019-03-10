package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	clusterName string
	timeout     = 30 * time.Second
)

var rootCmd = &cobra.Command{
	Use:              "sind",
	Short:            "Easily create swarm clusters on a docker host using swarm in docker.",
	TraverseChildren: true,
}

func init() {
	rootCmd.Flags().StringVarP(&clusterName, "cluster", "c", "sind_default", "Cluster name.")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "t", timeout, "Command timeout.")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func fail(pattern string, values ...interface{}) {
	fmt.Printf(pattern, values...)
	os.Exit(1)
}
