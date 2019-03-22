package cli

import (
	"fmt"
	"runtime"

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
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Go version: %s\n", runtime.Version()[2:])
}
