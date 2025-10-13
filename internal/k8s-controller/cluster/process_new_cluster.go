package cluster

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	
	"github.com/vlourd/galera-k8s-operator/internal/entities"
	clusterv1 "github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1"
	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
)

func (c *Controller) processNewCluster(ctx context.Context, state *clusterv1.ClusterTargetState) entities.ReconcileResult {
	nodes, err := c.s.GetNodesByCluster(ctx, state.Name)
	if err != nil {
		return entities.ReconcileResult{Error: err}
	}

	// creating first node
	if len(nodes) == 0 {
		c.l.Debug("creating primary node")

		primaryNode := nodev1.NodeTargetState{
			Version: nodev1.Version,
			Kind:    nodev1.Kind,
			Name:    fmt.Sprintf("%s-%s", state.Name, "node-0"),
			Spec: nodev1.Spec{
				ClusterName: state.Name,
				Image:       state.Spec.Image,
				AgentTargetState: nodev1.AgentTargetState{
					GatherHB:         true,
					MariaDBStarted:   true,
					GaleraNewCluster: true,
					NodeRole:         "primary",
					Galera:           nodev1.GaleraTargetState{},
					Tasks:            nil,
					Configuration:    nil,
				},
			},
		}
		res, err := primaryNode.ToGeneralTargetState()
		if err != nil {
			return entities.ReconcileResult{Error: err}
		}

		err = c.s.CreateResource(res)
		if err != nil {
			return entities.ReconcileResult{Error: err}
		}

		return entities.ReconcileResult{}
	}

	c.l.Debug("primary node exists, checking others")

	activeNodes := 0
	primaryNodes := 0
	atLeastOneIsntActive := false

	for _, node := range nodes {
		spew.Dump("NODE "+node.Name, node)
		if node.Spec.AgentTargetState.NodeRole == "primary" {
			primaryNodes++
		}

		if node.Spec.Status != entities.NodeStatusActive {
			c.l.Debug("node is not active... reqeue")
			atLeastOneIsntActive = true
		}
		activeNodes += 1
	}

	if primaryNodes == 0 || primaryNodes > 1 {
		c.l.Debug("more than one primary node exists")
		return entities.ReconcileResult{}
	}

	if atLeastOneIsntActive {
		c.l.Debug("at least one node is not active")
		return entities.ReconcileResult{}
	}

	c.l.Debug("there are more than 1 active node")

	if activeNodes < state.Spec.Replicas {
		c.l.Debug("creating secondary node")

		secondaryNode := nodev1.NodeTargetState{
			Version: nodev1.Version,
			Kind:    nodev1.Kind,
			Name:    fmt.Sprintf("node-%d", activeNodes+1),
			Spec: nodev1.Spec{
				ClusterName: state.Name,
				Image:       state.Spec.Image,
				Status:      entities.NodeStatusCreating,
				AgentTargetState: nodev1.AgentTargetState{
					GatherHB:         true,
					MariaDBStarted:   false,
					GaleraNewCluster: false,
					NodeRole:         "secondary",
					Galera:           nodev1.GaleraTargetState{},
					Tasks:            nil,
					Configuration:    nil,
				},
			},
		}
		res, err := secondaryNode.ToGeneralTargetState()
		if err != nil {
			return entities.ReconcileResult{Error: err}
		}

		err = c.s.CreateResource(res)
		if err != nil {
			return entities.ReconcileResult{Error: err}
		}

		return entities.ReconcileResult{}
	}

	c.l.Debug("all nodes are active")

	return entities.ReconcileResult{Reason: "unimplemented"}
}
