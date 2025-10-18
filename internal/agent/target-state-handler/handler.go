package target_state_handler

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent"
	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/agent/mariadb/manager"
)

type MariaDBManager interface {
	RunMariaDB(ctx context.Context, opts ...entities.RunOption) error
	StopMariaDB(ctx context.Context) error
	GetState() manager.State
}

type HBCollector interface {
	Collect(ctx context.Context) (*entities.State, error)
}

type ReconcilerOpts struct {
	l       *zap.Logger
	manager MariaDBManager
	hb      HBCollector
}

type Option func(*ReconcilerOpts)

func WithLogger(logger *zap.Logger) Option {
	return func(r *ReconcilerOpts) {
		r.l = logger
	}
}

func WithManager(m MariaDBManager) Option {
	return func(r *ReconcilerOpts) {
		r.manager = m
	}
}

func WithHB(hb HBCollector) Option {
	return func(r *ReconcilerOpts) {
		r.hb = hb
	}
}

type Reconciler struct {
	l       *zap.Logger
	manager agent.MariaDBManager
	hb      HBCollector
}

func New(opts ...Option) *Reconciler {
	params := &ReconcilerOpts{}
	for _, opt := range opts {
		opt(params)
	}

	return &Reconciler{
		l:       params.l,
		manager: params.manager,
		hb:      params.hb,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, ts *entities.TargetState) error {
	var err error
	r.l.Info("reconciling target state")
	defer func() {
		r.l.Info("reconcile complete")
	}()

	if ts == nil {
		return errors.New("nil target state")
	}

	state, err := r.hb.Collect(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to collect hb state")
	}
	r.l.Debug("got target state", zap.Any("state", state))

	if ts.GaleraNewCluster && ts.NodeRole == entities.RolePrimary {
		err = r.NewCluster(ctx, ts, state)
		if err != nil {
			return err
		}
		return nil
	}

	if ts.MariaDBStarted && ts.NodeRole == entities.RoleSecondary {
		err = r.JoinCluster(ctx, ts, state)
		if err != nil {
			return err
		}
	}

	return nil
}
