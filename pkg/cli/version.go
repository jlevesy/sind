package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "unknown"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints sind version.",
		Run:   runVersion,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(version)
}
