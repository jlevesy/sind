package internal

import "fmt"

const (
	clusterNameLabel = "com.sind.cluster.name"
	clusterRoleLabel = "com.sind.cluster.role"

	nodeRolePrimary = "primary"
	nodeRoleManager = "manager"
	nodeRoleWorker  = "worker"
)

func primaryNodeLabel() string {
	return fmt.Sprintf("%s=%s", clusterRoleLabel, nodeRolePrimary)
}

func clusterLabel(name string) string {
	return fmt.Sprintf("%s=%s", clusterNameLabel, name)
}
