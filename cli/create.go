package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new swarm cluster.",
	Run:   runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	fmt.Println("Hello world !")
}
