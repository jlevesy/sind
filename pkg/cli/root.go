package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

var (
	clusterName    string
	timeout        time.Duration
	nonInteractive bool
)

var rootCmd = &cobra.Command{
	Use:              "sind",
	Short:            "Easily create swarm clusters on a docker host using swarm in docker.",
	TraverseChildren: true,
	PreRun: func(*cobra.Command, []string) {
		disgo.SetTerminalOptions(disgo.WithInteractive(!nonInteractive))
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&clusterName, "cluster", "c", "default", "Cluster name.")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Command timeout.")
	rootCmd.PersistentFlags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Non interactive mode.")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func fail(err error) {
	disgo.Errorln(style.Failure(err))
	os.Exit(1)
}
