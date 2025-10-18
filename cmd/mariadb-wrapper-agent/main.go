package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vlourd/galera-k8s-operator/internal/agent"
	"github.com/vlourd/galera-k8s-operator/internal/agent/config"
	hbCollector "github.com/vlourd/galera-k8s-operator/internal/agent/hb-collector"
	"github.com/vlourd/galera-k8s-operator/internal/agent/mariadb/manager"
	targetStateHandler "github.com/vlourd/galera-k8s-operator/internal/agent/target-state-handler"
	targetStateManager "github.com/vlourd/galera-k8s-operator/internal/agent/target-state-manager"
)

//const (
//	//mariadbExecutable = `./waiter.sh`
//	taskManagerURL    = "task-manager.default.svc.cluster.local"
//	mariadbExecutable = "docker-entrypoint.sh"
//	galeraConfPath    = "/etc/mysql/conf.d/galera.cnf"
//)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enderCh := make(chan os.Signal, 1)
	signal.Notify(enderCh, syscall.SIGTERM, syscall.SIGINT)

	logger := initLogger()
	defer logger.Sync()

	conf, err := config.ConfigFromEnv()
	if err != nil {
		logger.Fatal("Error loading config", zap.Error(err))
	}

	mariadbManager := manager.New(logger)
	defer mariadbManager.StopMariaDB(ctx)

	taskManagerURL := conf.StateAPIURL
	hbCollector := hbCollector.New(
		hbCollector.WithLogger(logger),
		hbCollector.WithMariaDBManager(mariadbManager),
		hbCollector.WithStateUrl(taskManagerURL),
		hbCollector.WithName(conf.Name),
		hbCollector.WithVersion(conf.Version),
	)

	reconciler := targetStateHandler.New(
		targetStateHandler.WithLogger(logger),
		targetStateHandler.WithManager(mariadbManager),
		targetStateHandler.WithHB(hbCollector),
	)
	_ = agent.NewNodeAgent("1", mariadbManager)

	stateManager := targetStateManager.NewStateManager(
		targetStateManager.WithAgentID(conf.Name),
		targetStateManager.WithServerURL(taskManagerURL),
		targetStateManager.WithLogger(logger),
		targetStateManager.WithReconciler(reconciler),
		targetStateManager.WithHBCollector(hbCollector),
	)

	go func() {
		err := hbCollector.Start(ctx)
		if err != nil {
			logger.Fatal("failed to start agent", zap.Error(err))
		}
	}()

	//err = mariadbManager.RunMariaDB(ctx, entities.WithGaleraNewCluster())
	//if err != nil {
	//	logger.Fatal("failed to start mariadb", zap.Error(err))
	//}

	go func() {
		err = stateManager.RunForever(ctx)
		if err != nil {
			logger.Fatal("failed to start state manager", zap.Error(err))
		}
	}()

	logger.Info("waiting for signal")
	<-enderCh
	logger.Warn("received signal, shutting down")
}

func initLogger() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()

	// Configure the encoder to show full path
	encoderCfg.EncodeCaller = zapcore.FullCallerEncoder

	// Create logger with the custom configuration
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		),
		zap.AddCaller(),
	)

	return logger
}
