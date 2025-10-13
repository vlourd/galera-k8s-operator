package bootstrap

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent"
	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/agent/tasks"
)

type BootstrapPrimaryPayload struct {
	ClusterNodes []string `json:"cluster_nodes"`
	NodeAddress  string   `json:"node_address"`
}

type BootstrapPrimaryNodeTask struct {
	tasks.GeneralTask
	Spec BootstrapPrimaryPayload
}

func Execute(ctx context.Context, l *zap.Logger, task *tasks.GeneralTask, manager agent.MariaDBManager) error {
	payload := BootstrapPrimaryPayload{}
	err := json.Unmarshal([]byte(task.Payload), &payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload")
	}

	needRestart, err := ConfigureGalera(l, &payload)
	if err != nil {
		return errors.Wrap(err, "failed to configure Galera")
	}

	if !needRestart {
		return nil
	}

	err = manager.StopMariaDB(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to stop mariadb")
	}

	err = manager.RunMariaDB(ctx, entities.WithGaleraNewCluster())
	if err != nil {
		return errors.Wrap(err, "failed to run mariadb")
	}

	return nil
}
