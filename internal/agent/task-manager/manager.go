package task_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent/tasks"
)

type TaskHandler interface {
	Handle(ctx context.Context, task *tasks.GeneralTask) error
}
type TaskManager struct {
	AgentID string

	serverUrl   string
	logger      *zap.Logger
	httpClient  *http.Client
	taskHandler TaskHandler
}

func NewTaskManager(agentID string, serverUrl string, taskHandler TaskHandler, logger *zap.Logger) *TaskManager {
	return &TaskManager{
		AgentID:     agentID,
		serverUrl:   serverUrl,
		logger:      logger,
		taskHandler: taskHandler,
		httpClient:  &http.Client{},
	}
}

func (m *TaskManager) GetNextTask(ctx context.Context) (*tasks.GeneralTask, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("http://%s/tasks/next", m.serverUrl),
		nil,
	)
	if err != nil {
		m.logger.Error("Failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add agent ID to headers or query params
	q := req.URL.Query()
	q.Add("agent_id", m.AgentID)
	req.URL.RawQuery = q.Encode()

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("Failed to fetch task", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		m.logger.Info("No tasks available")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Unexpected status code", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task tasks.GeneralTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		m.logger.Error("Failed to decode task", zap.Error(err))
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	m.logger.Info("Received new task",
		zap.String("task_id", task.ID),
		zap.String("task_name", task.Name),
	)

	return &task, nil
}

func (m *TaskManager) RunForever(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			func() {
				ticker.Stop()
				defer ticker.Reset(5 * time.Second)
				task, err := m.GetNextTask(ctx)
				if err != nil {
					m.logger.Error("Failed to get next task", zap.Error(err))

				}

				taskUUID, _ := uuid.NewRandom()
				task = &tasks.GeneralTask{
					ID:      taskUUID.String(),
					Name:    "BootstrapPrimaryNodeTask",
					Payload: `{"cluster_nodes":["10.0.0.1","10.0.0.2","10.0.0.3"],"node_address":"10.0.0.1"}`,
				}
				m.logger.Info("Got next task", zap.String("task_id", task.ID), zap.String("task_name", task.Name))
				err = m.taskHandler.Handle(ctx, task)
				if err != nil {
					m.logger.Error("Failed to handle task", zap.Error(err))
				}
			}()

			// execute next task
		case <-ctx.Done():
			m.logger.Debug("TaskManager has stopped, context canceled")
			return nil
		}
	}
}
