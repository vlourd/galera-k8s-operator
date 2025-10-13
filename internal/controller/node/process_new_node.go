package node

import (
	"context"
	"errors"

	"go.uber.org/zap"

	v1 "github.com/vlourd/galera-k8s-operator/internal/entities"
	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
)

func (c *Controller) processNewNode(ctx context.Context, state nodev1.NodeTargetState) v1.ReconcileResult {
	c.l.Info("reconciling new node", zap.String("node", state.Name))
	if state.Spec.Heartbeat.Timestamp.IsZero() {
		c.l.Info("heartbeat timestamp is zero")
		return v1.ReconcileResult{Reason: "heartbeat timestamp is zero"}
	}

	if state.Spec.AgentTargetState.NodeRole == "primary" {
		c.l.Info("node is primary", zap.String("node", state.Name))
		return c.processNewPrimary(ctx, state)
	}

	if state.Spec.AgentTargetState.NodeRole == "secondary" {
		c.l.Info("node is secondary", zap.String("node", state.Name))
		return c.processNewSecondary(ctx, state)
	}

	c.l.Info("node has invalid role", zap.String("node", state.Name))
	return v1.ReconcileResult{Error: errors.New("invalid NodeRole")}
}

func (c *Controller) processNewPrimary(ctx context.Context, state nodev1.NodeTargetState) v1.ReconcileResult {
	c.l.Info("processing new primary node", zap.String("node", state.Name))
	if state.Spec.Heartbeat.Prepared && state.Spec.Heartbeat.WsrepInfo.WsrepClusterStatus == "Primary" {
		return v1.ReconcileResult{Success: true}
	}

	state.Spec.AgentTargetState = nodev1.AgentTargetState{
		GatherHB:         true,
		MariaDBStarted:   true,
		GaleraNewCluster: true,
		NodeRole:         "primary",
	}

	genState, err := state.ToGeneralTargetState()
	if err != nil {
		return v1.ReconcileResult{Error: err}
	}

	err = c.s.UpdateGeneralResource(genState)
	if err != nil {
		return v1.ReconcileResult{Error: err}
	}

	return v1.ReconcileResult{Success: false, Reason: "requeued"}
}

func (c *Controller) processNewSecondary(ctx context.Context, state nodev1.NodeTargetState) v1.ReconcileResult {
	c.l.Info("processing new secondary node", zap.String("node", state.Name))
	if state.Spec.Heartbeat.Prepared && state.Spec.Heartbeat.WsrepInfo.WsrepClusterStatus == "Primary" {
		return v1.ReconcileResult{Success: true}
	}

	nodes, err := c.s.GetNodesByCluster(ctx, state.Spec.ClusterName)
	if err != nil {
		return v1.ReconcileResult{Error: err}
	}
	c.l.Debug("all nodes", zap.Any("nodes", nodes))

	nodeIPs := make([]string, 0)
	for _, node := range nodes {
		if node.Spec.Heartbeat.IPAddress == "" {
			c.l.Info("node has no IP address", zap.String("node", node.Name))
			return v1.ReconcileResult{Error: errors.New("node has no IP address")}
		}

		nodeIPs = append(nodeIPs, node.Spec.Heartbeat.IPAddress)
	}

	state.Spec.AgentTargetState.Galera.IncomingAddresses = nodeIPs
	state.Spec.AgentTargetState.MariaDBStarted = true
	state.Spec.AgentTargetState.GaleraNewCluster = false

	getRes, err := state.ToGeneralTargetState()
	if err != nil {
		return v1.ReconcileResult{Error: err}
	}

	err = c.s.UpdateGeneralResource(getRes)
	if err != nil {
		return v1.ReconcileResult{Error: err}
	}
	c.l.Info("target state of secondary node has been updated", zap.String("node", state.Name))

	return v1.ReconcileResult{Success: false, Reason: "requeued"}
}
