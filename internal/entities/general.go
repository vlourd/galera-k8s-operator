package entities

import (
	"encoding/json"
	"fmt"
)

const (
	ClusterStatusCreating = "CREATING"
	ClusterStatusUpdating = "UPDATING"
	ClusterStatusToDelete = "DELETING"
	ClusterStatusActive   = "ACTIVE"
	ClusterStatusDeleted  = "REMOVED"
	ClusterStatusError    = "ERROR"

	NodeStatusCreating = "CREATING"
	NodeStatusUpdating = "UPDATING"
	NodeStatusToDelete = "DELETING"
	NodeStatusActive   = "ACTIVE"
	NodeStatusDeleted  = "REMOVED"
	NodeStatusError    = "ERROR"

	ReconcileStatusActive  = "ACTIVE"
	ReconcileStatusHealing = "HEALING"
)

type GeneralTargetState struct {
	Version string `json:"version"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`

	Spec json.RawMessage `json:"spec"`
	Meta json.RawMessage `json:"meta"`
}

// Copy creates a deep copy of the GeneralTargetState using JSON marshaling
func (g *GeneralTargetState) Copy() (GeneralTargetState, error) {
	// Marshal the original struct to JSON
	data, err := json.Marshal(g)
	if err != nil {
		return GeneralTargetState{}, fmt.Errorf("failed to marshal for copy: %w", err)
	}

	// Unmarshal into a new struct
	var dCopy GeneralTargetState
	if err := json.Unmarshal(data, &dCopy); err != nil {
		return GeneralTargetState{}, fmt.Errorf("failed to unmarshal for copy: %w", err)
	}

	return dCopy, nil
}

// MustCopy creates a deep copy and panics if there's an error
// Useful for cases where you're certain copying will succeed
func (g *GeneralTargetState) MustCopy() GeneralTargetState {
	dCopy, err := g.Copy()
	if err != nil {
		panic(fmt.Sprintf("GeneralTargetState copy failed: %v", err))
	}
	return dCopy
}

type ReconcileResult struct {
	Success bool
	Reason  string
	Error   error
}
