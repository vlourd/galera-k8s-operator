package agent

import (
	"context"

	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
)

type Agent struct {
	ID             string
	MariaDBManager MariaDBManager
}

func NewNodeAgent(id string, mariaManager MariaDBManager) *Agent {
	return &Agent{
		ID:             id,
		MariaDBManager: mariaManager,
	}
}

type MariaDBManager interface {
	RunMariaDB(ctx context.Context, opts ...entities.RunOption) error
	StopMariaDB(ctx context.Context) error
}
