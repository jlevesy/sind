package internal

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/api/types"
	"github.com/jlevesy/sind/pkg/sind"
	"github.com/ullaakut/disgo/style"
)

// RenderCluster renders cluster informations to given output.
func RenderCluster(out io.Writer, cluster sind.ClusterStatus) {
	wr := tabwriter.NewWriter(out, 4, 8, 2, '\t', 0)
	defer wr.Flush()
	fmt.Fprintf(
		wr,
		"Name: %s\tStatus: %s\tManagers: %s\t Workers: %s\t\n",
		style.Important(cluster.Name),
		style.Important(status(cluster)),
		style.Important(fmt.Sprintf("%d/%d", cluster.ManagersRunning, cluster.Managers)),
		style.Important(fmt.Sprintf("%d/%d", cluster.WorkersRunning, cluster.Workers)),
	)
	fmt.Fprintf(wr, "ID\tImage\tRole\tStatus\tIPs\t\n")
	fmt.Fprintf(wr, "--\t-----\t----\t------\t---\t\n")
	for _, node := range cluster.Nodes {
		fmt.Fprintf(
			wr,
			"%s\t%s\t%s\t%s\t%s\t\n",
			node.ID[0:11],
			node.Image,
			clusterRole(node),
			node.Status,
			nodeIP(node),
		)
	}
}

func clusterRole(node types.Container) string {
	return node.Labels["com.sind.cluster.role"]
}

func nodeIP(node types.Container) string {
	if node.NetworkSettings == nil {
		return ""
	}

	var IPs []string

	for _, net := range node.NetworkSettings.Networks {
		IPs = append(IPs, net.IPAddress)
	}

	return strings.Join(IPs, ",")
}
