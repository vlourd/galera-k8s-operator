package task_handler

import (
	"context"
	
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent"
	"github.com/vlourd/galera-k8s-operator/internal/agent/tasks"
	"github.com/vlourd/galera-k8s-operator/internal/agent/tasks/bootstrap"
)

type TaskHandler struct {
	logger         *zap.Logger
	mariadbManager agent.MariaDBManager
}

func New(logger *zap.Logger, mariadbManager agent.MariaDBManager) *TaskHandler {
	return &TaskHandler{logger: logger, mariadbManager: mariadbManager}
}

func (t *TaskHandler) Handle(ctx context.Context, task *tasks.GeneralTask) error {
	var err error

	switch task.Name {
	case "BootstrapPrimaryNodeTask":
		err = bootstrap.Execute(ctx, t.logger, task, t.mariadbManager)

	default:
		return errors.New("unknown task name")
	}

	if err != nil {
		t.logger.Error("failed to execute task", zap.Error(err))
	}
	return nil
}
