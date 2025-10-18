package cluster

import (
	"context"

	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
	clusterv1 "github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1"
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
	resources, err := c.s.ListGeneralResourcesByKind(clusterv1.Kind)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (c *Controller) Reconcile(ctx context.Context, resource entities.GeneralTargetState) entities.ReconcileResult {
	c.l.Info("Reconciling Cluster", zap.Any("resource", resource))
	res := entities.ReconcileResult{}
	cluster := &clusterv1.ClusterTargetState{}
	err := cluster.FromGeneralTargetState(resource)
	if err != nil {
		return entities.ReconcileResult{Error: err}
	}

	switch cluster.Spec.Status {
	case entities.ClusterStatusCreating:
		res = c.processNewCluster(ctx, cluster)
	}

	if !res.Success {
		return res
	}

	return c.finalize(ctx, res, cluster)
}

func (c *Controller) finalize(ctx context.Context, res entities.ReconcileResult, state *clusterv1.ClusterTargetState) entities.ReconcileResult {
	return res
}
