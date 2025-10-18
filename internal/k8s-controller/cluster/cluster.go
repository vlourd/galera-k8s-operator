package cluster

import (
	"context"
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
	clusterv1 "github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1"
	"github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1/k8s"
	inMemoryStorage "github.com/vlourd/galera-k8s-operator/internal/in-memory-storage"
	"k8s.io/client-go/dynamic"
)

type Controller struct {
	l   *zap.Logger
	s   *inMemoryStorage.Storage
	cli dynamic.Interface
}

func NewController(l *zap.Logger, s *inMemoryStorage.Storage, cli dynamic.Interface) *Controller {
	return &Controller{
		l:   l,
		s:   s,
		cli: cli,
	}
}

func (c *Controller) GetResources(ctx context.Context) ([]entities.GeneralTargetState, error) {
	resources := make([]entities.GeneralTargetState, 0)

	apiGroup := schema.GroupVersionResource{
		Group:    "operator.vlourd",
		Version:  "betav1",
		Resource: "galeraclusters",
	}

	k8sResource := c.cli.Resource(apiGroup)
	c.l.Debug("getting resources", zap.Any("resources", k8sResource))

	k8sUnstructuredResources, err := c.cli.Resource(apiGroup).Namespace("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Error("failed to list k8s resources", zap.Error(err), zap.Any("k8s_resource", k8sUnstructuredResources))
		return nil, errors.Wrap(err, "failed to list k8s resources")
	}

	for _, k8sUnstructuredResource := range k8sUnstructuredResources.Items {
		c.l.Info("k8s resource found", zap.Any("resource", k8sUnstructuredResource.UnstructuredContent()))
		k8sCluster := k8s.GaleraCluster{}
		cont, err := k8sUnstructuredResource.MarshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal k8s resource to JSON")
		}
		c.l.Info("k8s cluster found", zap.Any("k8s_cluster", string(cont)))

		err = json.Unmarshal(cont, &k8sCluster)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal k8s resource to GaleraCluster")
		}

		spew.Dump(k8sCluster)

		c.l.Info("k8s resource marshaled", zap.Any("resource", k8sCluster))
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
