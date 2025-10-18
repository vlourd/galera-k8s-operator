package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	cntrlManager "github.com/vlourd/galera-k8s-operator/internal/controller-manager"
	"github.com/vlourd/galera-k8s-operator/internal/controller/cluster"
	"github.com/vlourd/galera-k8s-operator/internal/controller/node"
	inMemoryStorage "github.com/vlourd/galera-k8s-operator/internal/in-memory-storage"
	stateApi "github.com/vlourd/galera-k8s-operator/internal/state-api"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enderCh := make(chan os.Signal, 1)
	signal.Notify(enderCh, syscall.SIGTERM, syscall.SIGINT)

	logger := initLogger()
	defer logger.Sync()

	storage := inMemoryStorage.NewStorage()
	controllerManager := cntrlManager.NewControllerManager(
		cntrlManager.WithLogger(logger),
		cntrlManager.WithStorage(storage),
	)

	stateServer := stateApi.NewServer(logger, storage)

	go func() {
		err := stateServer.Start(ctx, "16543")
		if err != nil {
			logger.Fatal("failed to start state server", zap.Error(err))
		}
	}()

	clusterController := cluster.NewController(logger, storage)
	nodeController := node.NewController(logger, storage)

	// registering controllers
	controllerManager.RegisterController("cluster", clusterController)
	controllerManager.RegisterController("node", nodeController)

	go func() {
		controllerManager.Run(ctx)
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
