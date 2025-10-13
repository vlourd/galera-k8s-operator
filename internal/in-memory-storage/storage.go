package in_memory_storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
	clusterv1 "github.com/vlourd/galera-k8s-operator/internal/entities/cluster/betav1"
	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
)

type Storage struct {
	gr map[string]*entities.GeneralTargetState

	mu sync.RWMutex
}

func (s *Storage) Init() error {
	replicas := 3
	cluster := clusterv1.ClusterTargetState{
		Version:   clusterv1.Version,
		Kind:      clusterv1.Kind,
		Name:      "my-galera-cluster",
		Namespace: "default",
		Spec: clusterv1.Spec{
			Replicas: replicas,
			Status:   entities.ClusterStatusCreating,
			Image:    "mariadb-op:latest",
		},
		Meta: clusterv1.Meta{
			//CreatedAt: "2023-01-01T00:00:00Z",
		},
	}

	generalRes, err := cluster.ToGeneralTargetState()
	if err != nil {
		return err
	}

	err = s.CreateResource(generalRes)
	if err != nil {
		return err
	}

	nodes := []nodev1.NodeTargetState{
		{
			Version: "betav1",
			Name:    "node-1",
			Kind:    nodev1.Kind,
			Spec: nodev1.Spec{
				ClusterName: "my-galera-cluster",
				Image:       "mariadb-op:latest",
				Status:      entities.NodeStatusCreating,
				AgentTargetState: nodev1.AgentTargetState{
					GatherHB:         true,
					MariaDBStarted:   true,
					NodeRole:         "primary",
					GaleraNewCluster: true,
				},
			},
			Meta: nodev1.Meta{},
		},
		//{
		//	Version: "betav1",
		//	Name:    "node-2",
		//	Kind:    nodev1.Kind,
		//	Spec: nodev1.Spec{
		//		ClusterName: "my-galera-cluster",
		//		Image:       "mariadb-op:latest",
		//		Status:      entities.NodeStatusCreating,
		//		AgentTargetState: nodev1.AgentTargetState{
		//			GatherHB:         true,
		//			MariaDBStarted:   false,
		//			NodeRole:         "secondary",
		//			GaleraNewCluster: false,
		//		},
		//	},
		//	Meta: nodev1.Meta{},
		//},
		//{
		//	Version: "betav1",
		//	Name:    "node-3",
		//	Kind:    nodev1.Kind,
		//	Spec: nodev1.Spec{
		//		ClusterName: "my-galera-cluster",
		//		Image:       "mariadb-op:latest",
		//		Status:      entities.NodeStatusCreating,
		//		AgentTargetState: nodev1.AgentTargetState{
		//			GatherHB:         true,
		//			MariaDBStarted:   false,
		//			NodeRole:         "secondary",
		//			GaleraNewCluster: false,
		//		},
		//	},
		//	Meta: nodev1.Meta{},
		//},
	}

	for _, node := range nodes {
		nodeRes, err := node.ToGeneralTargetState()
		if err != nil {
			return err
		}

		err = s.CreateResource(nodeRes)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewStorage() *Storage {
	s := &Storage{
		gr: make(map[string]*entities.GeneralTargetState),
		mu: sync.RWMutex{},
	}
	_ = s.Init()
	return s
}

func (s *Storage) CreateResource(entity entities.GeneralTargetState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dCopy, err := entity.Copy()
	if err != nil {
		return err
	}

	kindKey := fmt.Sprintf("/%s/%s", entity.Kind, entity.Name)
	s.gr[kindKey] = &dCopy
	return nil
}

func (s *Storage) GetGeneralResource(name, kind string) (entities.GeneralTargetState, error) {
	if name == "" || kind == "" {
		return entities.GeneralTargetState{}, errors.New("invalid parameter")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("/%s/%s", kind, name)
	if _, ok := s.gr[key]; !ok {
		return entities.GeneralTargetState{}, nil
	}

	dCopy, err := s.gr[key].Copy()
	if err != nil {
		return entities.GeneralTargetState{}, err
	}

	return dCopy, nil
}

func (s *Storage) UpdateGeneralResource(state entities.GeneralTargetState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cState, err := state.Copy()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("/%s/%s", state.Kind, state.Name)
	if _, ok := s.gr[key]; !ok {
		return errors.New("resource not found")
	}

	s.gr[key] = &cState
	return nil
}

func (s *Storage) DeleteGeneralResource(name, kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("/%s/%s", kind, name)
	delete(s.gr, key)
	return nil
}

func (s *Storage) ListGeneralResourcesByKind(kind string) ([]entities.GeneralTargetState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]entities.GeneralTargetState, 0, len(s.gr))
	for _, v := range s.gr {
		if v.Kind != kind {
			continue
		}

		copyV, err := v.Copy()
		if err != nil {
			return nil, err
		}
		list = append(list, copyV)
	}

	return list, nil
}
