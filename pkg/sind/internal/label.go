package internal

import "fmt"

const (
	clusterNameLabel = "com.sind.cluster.name"
	clusterRoleLabel = "com.sind.cluster.role"

	primaryNodeLabel = "com.sind.cluster.role=primary"
)

func clusterLabel(name string) string {
	return fmt.Sprintf("%s=%s", clusterNameLabel, name)
}
