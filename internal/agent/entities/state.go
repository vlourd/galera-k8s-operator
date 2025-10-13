package entities

import "github.com/vlourd/galera-k8s-operator/internal/agent/tasks"

const (
	PreparedFlagPath = "/etc/agent/prepared"

	RolePrimary   = "primary"
	RoleSecondary = "secondary"
)

type TargetState struct {
	GatherHB         bool                `json:"gather_hb"`
	MariaDBStarted   bool                `json:"mariadb_started"`
	GaleraNewCluster bool                `json:"galera_new_cluster"`
	NodeRole         string              `json:"role"`
	Galera           GaleraTargetState   `json:"galera"`
	Tasks            []tasks.GeneralTask `json:"tasks,omitempty"`
	Configuration    map[string]string   `json:"configuration,omitempty"`
}

type GaleraTargetState struct {
	ClusterSize       int      `json:"cluster_size"`
	IncomingAddresses []string `json:"incoming_addresses"`
}
