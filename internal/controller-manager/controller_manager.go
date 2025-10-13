package controller_manager

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/entities"
	inMemoryStorage "github.com/vlourd/galera-k8s-operator/internal/in-memory-storage"
)

type Controller interface {
	GetResources(ctx context.Context) ([]entities.GeneralTargetState, error)
	Reconcile(ctx context.Context, resource entities.GeneralTargetState) entities.ReconcileResult
}

type ControllerManagerOptions struct {
	storage *inMemoryStorage.Storage
	l       *zap.Logger
}

type Option func(*ControllerManagerOptions)

func WithStorage(storage *inMemoryStorage.Storage) Option {
	return func(o *ControllerManagerOptions) {
		o.storage = storage
	}
}

func WithLogger(l *zap.Logger) Option {
	return func(o *ControllerManagerOptions) {
		o.l = l
	}
}

type ControllerManager struct {
	controllers map[string]Controller
	l           *zap.Logger
	storage     *inMemoryStorage.Storage

	mu sync.Mutex
}

func NewControllerManager(opts ...Option) *ControllerManager {
	params := &ControllerManagerOptions{}
	for _, opt := range opts {
		opt(params)
	}

	return &ControllerManager{
		controllers: make(map[string]Controller),
		l:           params.l,
		storage:     params.storage,
	}
}

func (c *ControllerManager) RegisterController(name string, controller Controller) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.controllers[name] = controller
}

func (c *ControllerManager) runController(ctx context.Context, controller Controller) error {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ticker.Stop()
			func() {
				defer ticker.Reset(10 * time.Second)
				err := c.doIteration(ctx, controller)
				if err != nil {
					c.l.Error("failed to do next iteration of controller", zap.Error(err))
				}
			}()
		}
	}
}

func (c *ControllerManager) doIteration(ctx context.Context, controller Controller) error {
	resources, err := controller.GetResources(ctx)
	if err != nil {
		return err
	}
	limiter := make(chan struct{}, 20)

	for _, res := range resources {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		limiter <- struct{}{}
		if err != nil {
			return err
		}

		go func() {
			defer func() { <-limiter }()
			result := controller.Reconcile(ctx, res)
			c.l.Info("reconcile result", zap.Any("result", result))
		}()
	}

	return nil
}

func (c *ControllerManager) Run(ctx context.Context) {
	c.l.Info("starting controllers")
	wg := sync.WaitGroup{}

	for name, ctrl := range c.controllers {
		wg.Add(1)
		c.l.Info("starting controller", zap.String("controller", name))
		go func() {
			defer wg.Done()
			err := c.runController(ctx, ctrl)
			if err != nil {
				c.l.Fatal("controller failed", zap.String("controller", name), zap.Error(err))
			}
		}()
	}

	wg.Wait()
}
