package dto

import (
	"github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1"
	"github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1/k8s"
)

func K8sClusterToBetav1Cluster(cluster k8s.GaleraCluster) betav1.ClusterTargetState {
	return betav1.ClusterTargetState{
		Version: cluster.APIVersion,
		Kind:    cluster.Kind,
		Name:    cluster.Name,

		Spec: betav1.Spec{
			Replicas: cluster.Spec.Replicas,
			Status:   "",
			Image:    cluster.Spec.Image,
		},

		Meta: betav1.Meta{},
	}
}
