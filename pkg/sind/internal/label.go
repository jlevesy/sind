package internal

import "fmt"

const (
	// ClusterNameLabel is the label containing the cluster name applied to resources of a cluster.
	ClusterNameLabel = "com.sind.cluster.name"

	// NodeRoleLabel is the label containing the cluster role applied to nodes (containers) of a cluster.
	NodeRoleLabel = "com.sind.cluster.role"
)

// Node roles.
const (
	NodeRolePrimary = "primary"
	NodeRoleManager = "manager"
	NodeRoleWorker  = "worker"
)

// PrimaryNodeLabel is the label applied to the primary node of a cluster.:
func PrimaryNodeLabel() string {
	return fmt.Sprintf("%s=%s", NodeRoleLabel, NodeRolePrimary)
}

// ClusterLabel returns the labels applied to each resource(network, container...) of a cluster.
func ClusterLabel(name string) string {
	return fmt.Sprintf("%s=%s", ClusterNameLabel, name)
}
