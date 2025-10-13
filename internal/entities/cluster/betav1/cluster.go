package betav1

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
)

const (
	Version = "betav1"
	Kind    = "Galera"
)

type ClusterTargetState struct {
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	Spec Spec `json:"spec"`
	Meta Meta `json:"meta"`
}

type Spec struct {
	Replicas int `json:"replicas"`
	Status   string
	Image    string
}

type Meta struct {
}

func (c *ClusterTargetState) FromGeneralTargetState(state entities.GeneralTargetState) error {
	cluster := &ClusterTargetState{
		Version: Version,
		Name:    state.Name,
		Kind:    Kind,
	}

	if err := json.Unmarshal(state.Spec, &cluster.Spec); err != nil {
		return errors.Wrap(err, "failed to unmarshal spec")
	}

	if err := json.Unmarshal(state.Meta, &cluster.Meta); err != nil {
		return errors.Wrap(err, "failed to unmarshal meta")
	}

	*c = *cluster
	return nil
}

func (c *ClusterTargetState) ToGeneralTargetState() (entities.GeneralTargetState, error) {
	spec, err := json.Marshal(&c.Spec)
	if err != nil {
		return entities.GeneralTargetState{}, errors.Wrap(err, "failed to marshal spec")
	}

	meta, err := json.Marshal(&c.Meta)
	if err != nil {
		return entities.GeneralTargetState{}, errors.Wrap(err, "failed to marshal meta")
	}

	return entities.GeneralTargetState{
		Name:    c.Name,
		Kind:    c.Kind,
		Version: Version,

		Spec: spec,
		Meta: meta,
	}, nil
}
