package node

import (
	"context"

	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
	inMemoryStorage "github.com/vlourd/galera-k8s-operator/internal/in-memory-storage"
)

type Controller struct {
	l *zap.Logger
	s *inMemoryStorage.Storage
}

func NewController(l *zap.Logger, s *inMemoryStorage.Storage) *Controller {
	return &Controller{
		l: l,
		s: s,
	}
}

func (c *Controller) GetResources(ctx context.Context) ([]entities.GeneralTargetState, error) {
	resources, err := c.s.ListGeneralResourcesByKind(nodev1.Kind)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (c *Controller) Reconcile(ctx context.Context, resource entities.GeneralTargetState) entities.ReconcileResult {
	c.l.Info("Reconciling Node", zap.Any("resource", resource))
	res := entities.ReconcileResult{}
	node := nodev1.NodeTargetState{}
	err := node.FromGeneralTargetState(resource)
	if err != nil {
		return entities.ReconcileResult{Error: err}
	}

	switch node.Spec.Status {
	case entities.NodeStatusCreating:
		res = c.processNewNode(ctx, node)
	}

	if !res.Success {
		return res
	}

	return c.finalize(ctx, res, &node)
}

func (c *Controller) finalize(ctx context.Context, res entities.ReconcileResult, state *nodev1.NodeTargetState) entities.ReconcileResult {
	c.l.Info("Finalizing Node", zap.Any("resource", state))
	newStatus := state.Spec.Status

	switch state.Spec.Status {
	case entities.NodeStatusCreating:
		newStatus = entities.NodeStatusActive
	}

	if newStatus == state.Spec.Status {
		return res
	}

	state.Spec.Status = newStatus
	genRes, err := state.ToGeneralTargetState()
	if err != nil {
		res.Error = err
	}

	err = c.s.UpdateGeneralResource(genRes)
	if err != nil {
		res.Error = err
	}

	return res
}
