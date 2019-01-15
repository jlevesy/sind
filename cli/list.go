package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:     "list",
		Short:   "List existing clusters.",
		Aliases: []string{"ls"},
		Run:     runList,
	}
)

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	store, err := NewStore()
	if err != nil {
		fail("unable to create store: %v\n", err)
	}

	clusters, err := store.List()
	if err != nil {
		fail("unable to list existing clusters: %v\n", err)
	}

	wr := tabwriter.NewWriter(os.Stdout, 4, 8, 0, '\t', 0)
	fmt.Fprintf(wr, "NAME\tSWARM DOCKER HOST\tDOCKER HOST\t\n")
	for _, cluster := range clusters {
		fmt.Fprintf(wr, "%s\t%s\t%s\t\n", cluster.Name, cluster.Cluster.DockerHost(), cluster.Host.Host)
	}
	wr.Flush()
	fmt.Printf("\nTotal: %d\n", len(clusters))
}
