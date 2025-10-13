package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	cntrlManager "github.com/vlourd/galera-k8s-operator/internal/controller-manager"
	inMemoryStorage "github.com/vlourd/galera-k8s-operator/internal/in-memory-storage"
	"github.com/vlourd/galera-k8s-operator/internal/k8s-controller/cluster"
	stateApi "github.com/vlourd/galera-k8s-operator/internal/state-api"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enderCh := make(chan os.Signal)
	signal.Notify(enderCh, syscall.SIGTERM, syscall.SIGINT)

	logger := initLogger()
	defer logger.Sync()

	// 1. Get kubeconfig
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	// 2. Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

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

	clusterController := cluster.NewController(logger, storage, dynamicClient)
	//nodeController := node.NewController(logger, storage)

	// registering controllers
	controllerManager.RegisterController("cluster", clusterController)
	//controllerManager.RegisterController("node", nodeController)

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
