package internal

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jlevesy/sind/pkg/sind"
)

// RenderClusterList renders a list of clusters in the given output.
func RenderClusterList(out io.Writer, clusters []sind.ClusterStatus) {
	wr := tabwriter.NewWriter(out, 4, 8, 2, '\t', 0)
	defer wr.Flush()
	fmt.Fprintf(wr, "\nName\tStatus\tManagers\tWorkers\t\n")
	fmt.Fprintf(wr, "----\t------\t--------\t-------\t\n")
	for _, cluster := range clusters {
		fmt.Fprintf(
			wr,
			"%s\t%s\t%d/%d\t%d/%d\t\n",
			cluster.Name,
			status(cluster),
			cluster.ManagersRunning,
			cluster.Managers,
			cluster.WorkersRunning,
			cluster.Workers,
		)
	}
}

func status(cluster sind.ClusterStatus) string {
	if cluster.ManagersRunning == 0 && cluster.WorkersRunning == 0 {
		return "Stopped"
	}

	if cluster.ManagersRunning == cluster.Managers && cluster.WorkersRunning == cluster.Workers {
		return "Running"
	}

	return "Unstable"
}
