package target_state_handler

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/agent/mariadb/manager"
	"github.com/vlourd/galera-k8s-operator/internal/agent/utils"
)

func (r *Reconciler) JoinCluster(ctx context.Context, ts *entities.TargetState, fact *entities.State) error {
	r.l.Info("checking if cluster is prepared", zap.String("state", fact.MariadbStatus), zap.Bool("prepared", fact.Prepared))
	r.l.Info("joining to the cluster")
	needRestart, err := r.configureGalera(ts)
	if err != nil {
		return errors.Wrap(err, "failed to configure Galera")
	}
	if !needRestart && fact.MariadbStatus == manager.StateRunning {
		return nil
	}

	err = r.manager.StopMariaDB(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to stop mariadb")
	}

	err = r.manager.RunMariaDB(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to run mariadb")
	}

	err = utils.TouchFile(entities.PreparedFlagPath)
	if err != nil {
		return errors.Wrap(err, "failed to touch prepared_flag_path")
	}

	return nil
}
