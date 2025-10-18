package manager

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
)

type State = string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateFailed  State = "failed"
	StateUnknown State = "unknown"
)

const mariadbExecutable = "docker-entrypoint.sh"

type MariaDBManager struct {
	State State
	PID   int

	logger *zap.Logger
	proc   *os.Process
	mu     sync.Mutex

	// started is a semaphore that guaranties finalizer is executed
	// before new process is started as it can result in setting wrong status
	// for mariadb process
	// it's unlocked only after finalizer finished its execution
	started sync.Mutex
}

func New(logger *zap.Logger) *MariaDBManager {
	return &MariaDBManager{State: StateUnknown, logger: logger}
}

func (m *MariaDBManager) RunMariaDB(ctx context.Context, opts ...entities.RunOption) error {
	m.started.Lock()
	m.mu.Lock()
	defer func() {
		go func() {
			m.finalize()
		}()
	}()
	defer m.mu.Unlock()

	if m.State == StateRunning {
		return nil
	}

	cmd := exec.CommandContext(ctx, mariadbExecutable)
	args := []string{"mariadbd"}
	runOpts := &entities.RunOptions{}
	for _, opt := range opts {
		opt(runOpts)
	}

	if runOpts.GaleraNewCluster {
		args = append(args, "--wsrep-new-cluster")
	}

	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "MYSQL_ROOT_PASSWORD=root")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start mariadb")
	}

	m.proc = cmd.Process
	m.State = StateRunning
	m.PID = cmd.Process.Pid

	return nil
}

func (m *MariaDBManager) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.State
}

func (m *MariaDBManager) finalize() {
	defer m.started.Unlock()
	state, err := m.proc.Wait()
	if err != nil {
		m.logger.Error("failed to wait for the process", zap.Error(err))
	}
	m.logger.Info("mariadb has been stopped", zap.String("exit_code", state.String()))

	m.mu.Lock()
	defer m.mu.Unlock()
	m.PID = 0
	m.State = StateStopped
}

func (m *MariaDBManager) StopMariaDB(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.State != StateRunning {
		return nil
	}

	err := m.proc.Signal(syscall.SIGTERM)
	if err != nil {
		m.logger.Error("failed to send SIGTERM, sending SIGKILL", zap.Error(err))

		// killing mariadb is wrong because it may have child processes
		// e.g. for replication slots
		// so better option is to kill everything which is connected to mariadb
		_ = m.proc.Kill()
	}
	return nil
}
