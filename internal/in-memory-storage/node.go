package in_memory_storage

import (
	"context"

	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
)

func (s *Storage) GetNodesByCluster(ctx context.Context, clusterName string) ([]nodev1.NodeTargetState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]nodev1.NodeTargetState, 0)

	for _, v := range s.gr {
		if v.Kind != nodev1.Kind {
			continue
		}

		node := nodev1.NodeTargetState{}
		err := node.FromGeneralTargetState(*v)
		if err != nil {
			return nil, err
		}

		if node.Spec.ClusterName != clusterName {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
