package betav1

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/agent/tasks"
	cpEntities "github.com/vlourd/galera-k8s-operator/internal/entities"
)

const (
	Version = "betav1"
	Kind    = "node"
)

type NodeTargetState struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`

	Spec Spec `json:"spec"`
	Meta Meta `json:"meta"`
}

func (n *NodeTargetState) GetVersion() string {
	return Version
}

func (n *NodeTargetState) FromGeneralTargetState(state cpEntities.GeneralTargetState) error {
	node := &NodeTargetState{
		Version: Version,
		Name:    state.Name,
		Kind:    Kind,
	}

	if err := json.Unmarshal(state.Spec, &node.Spec); err != nil {
		return errors.Wrap(err, "failed to unmarshal spec")
	}

	if err := json.Unmarshal(state.Meta, &node.Meta); err != nil {
		return errors.Wrap(err, "failed to unmarshal meta")
	}

	*n = *node
	return nil
}

func (n *NodeTargetState) ToGeneralTargetState() (cpEntities.GeneralTargetState, error) {
	spec, err := json.Marshal(&n.Spec)
	if err != nil {
		return cpEntities.GeneralTargetState{}, errors.Wrap(err, "failed to marshal spec")
	}

	meta, err := json.Marshal(&n.Meta)
	if err != nil {
		return cpEntities.GeneralTargetState{}, errors.Wrap(err, "failed to marshal meta")
	}

	return cpEntities.GeneralTargetState{
		Name:    n.Name,
		Kind:    n.Kind,
		Version: Version,

		Spec: spec,
		Meta: meta,
	}, nil
}

type Spec struct {
	ClusterName      string           `json:"cluster_name"`
	Image            string           `json:"image"`
	Status           string           `json:"status"`
	AgentTargetState AgentTargetState `json:"agent_target_state"`
	Heartbeat        entities.State   `json:"heartbeat"`
}

type Meta struct {
}

type AgentTargetState struct {
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
