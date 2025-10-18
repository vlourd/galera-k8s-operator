package target_state_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
)

type TargetStateReconciler interface {
	Reconcile(ctx context.Context, ts *entities.TargetState) error
}

type HBCollector interface {
	Collect(ctx context.Context) (*entities.State, error)
}

type StateManagerOpts struct {
	agentID    string
	serverURL  string
	logger     *zap.Logger
	reconciler TargetStateReconciler
	hb         HBCollector
}

type Option func(*StateManagerOpts)

func WithAgentID(agentID string) Option {
	return func(opts *StateManagerOpts) {
		opts.agentID = agentID
	}
}

func WithServerURL(serverURL string) Option {
	return func(opts *StateManagerOpts) {
		opts.serverURL = serverURL
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(opts *StateManagerOpts) {
		opts.logger = logger
	}
}

func WithReconciler(reconciler TargetStateReconciler) Option {
	return func(opts *StateManagerOpts) {
		opts.reconciler = reconciler
	}
}

func WithHBCollector(hb HBCollector) Option {
	return func(opts *StateManagerOpts) {
		opts.hb = hb
	}
}

type StateManager struct {
	AgentID    string
	serverUrl  string
	logger     *zap.Logger
	httpClient *http.Client
	reconciler TargetStateReconciler
	hb         HBCollector
}

func NewStateManager(opts ...Option) *StateManager {
	params := StateManagerOpts{}

	for _, opt := range opts {
		opt(&params)
	}

	return &StateManager{
		AgentID:    params.agentID,
		serverUrl:  params.serverURL,
		logger:     params.logger,
		httpClient: &http.Client{},
		reconciler: params.reconciler,
		hb:         params.hb,
	}
}

func (m *StateManager) GetTargetState(ctx context.Context) (*entities.TargetState, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("http://%s/target-state", m.serverUrl),
		nil,
	)
	if err != nil {
		m.logger.Error("Failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add agent ID to headers or query params
	q := req.URL.Query()
	q.Add("name", m.AgentID)
	req.URL.RawQuery = q.Encode()

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("Failed to fetch target state", zap.Error(err))
		return nil, fmt.Errorf("failed to terget state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		m.logger.Info("No target state available")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Unexpected status code", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task entities.TargetState
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		m.logger.Error("Failed to decode target state", zap.Error(err))
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	m.logger.Info("Received target state")

	return &task, nil
}

func (m *StateManager) RunForever(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			func() {
				ticker.Stop()
				defer ticker.Reset(5 * time.Second)
				targetState, err := m.GetTargetState(ctx)
				if err != nil {
					m.logger.Error("Failed to get target state", zap.Error(err))

				}

				//taskUUID, _ := uuid.NewRandom()
				//targetState = &entities.TargetState{
				//	GatherHB:         true,
				//	MariaDBStarted:   true,
				//	GaleraNewCluster: true,
				//	NodeRole:         entities.RolePrimary,
				//	Galera: entities.GaleraTargetState{
				//		ClusterSize:       3,
				//		IncomingAddresses: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
				//	},
				//}
				tsData, err := json.Marshal(targetState)
				//task = &tasks.GeneralTask{
				//	ID:      taskUUID.String(),
				//	Name:    "BootstrapPrimaryNodeTask",
				//	Payload: `{"cluster_nodes":["10.0.0.1","10.0.0.2","10.0.0.3"],"node_address":"10.0.0.1"}`,
				//}
				m.logger.Info("Got target state", zap.String("target_state", string(tsData)))
				err = m.reconciler.Reconcile(ctx, targetState)
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
