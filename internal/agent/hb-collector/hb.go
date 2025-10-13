package hb_collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	
	"github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/agent/mariadb/manager"
	"github.com/vlourd/galera-k8s-operator/internal/agent/mariadb/service"
	"github.com/vlourd/galera-k8s-operator/internal/agent/utils"
)

type MariaDBManager interface {
	RunMariaDB(ctx context.Context, opts ...entities.RunOption) error
	StopMariaDB(ctx context.Context) error
	GetState() manager.State
}

type HBCollectorOpts struct {
	logger         *zap.Logger
	httpClient     *http.Client
	mariadbManager MariaDBManager
	stateUrl       string
	name           string
	version        string
}

type Option func(*HBCollectorOpts)

func WithLogger(logger *zap.Logger) Option {
	return func(opts *HBCollectorOpts) {
		opts.logger = logger
	}
}

func WithHttpClient(httpClient *http.Client) Option {
	return func(opts *HBCollectorOpts) {
		opts.httpClient = httpClient
	}
}

func WithMariaDBManager(mariadbManager MariaDBManager) Option {
	return func(opts *HBCollectorOpts) {
		opts.mariadbManager = mariadbManager
	}
}

func WithStateUrl(stateUrl string) Option {
	return func(opts *HBCollectorOpts) {
		opts.stateUrl = stateUrl
	}
}

func WithName(name string) Option {
	return func(opts *HBCollectorOpts) {
		opts.name = name
	}
}

func WithVersion(version string) Option {
	return func(opts *HBCollectorOpts) {
		opts.version = version
	}
}

type HBCollector struct {
	mariadbManager MariaDBManager
	logger         *zap.Logger
	httpClient     *http.Client
	stateURL       string
	lastState      *entities.State
	name           string
	version        string

	mu sync.Mutex
}

func New(opts ...Option) *HBCollector {
	params := &HBCollectorOpts{}
	for _, opt := range opts {
		opt(params)
	}

	if params.httpClient == nil {
		params.httpClient = &http.Client{}
	}

	return &HBCollector{
		logger:         params.logger,
		httpClient:     params.httpClient,
		mariadbManager: params.mariadbManager,
		stateURL:       params.stateUrl,
		name:           params.name,
		version:        params.version,
	}
}

func (c *HBCollector) Start(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ticker.Stop()
			func() {
				c.logger.Debug("start collecting heartbeat", zap.String("name", c.name))
				defer ticker.Reset(10 * time.Second)
				state, err := c.collect(ctx)
				if err != nil {
					c.logger.Error("failed to collect heartbeat", zap.Error(err))

					return
				}

				err = c.send(ctx, state)
				if err != nil {
					c.logger.Error("failed to send heartbeat", zap.Error(err))
				}
			}()
		default:
		}
	}
}

func (c *HBCollector) Collect(ctx context.Context) (*entities.State, error) {
	c.mu.Lock()
	lastState := c.lastState
	c.mu.Unlock()

	if lastState == nil {
		_, err := c.collect(ctx)
		if err != nil {
			c.logger.Error("failed to collect heartbeat", zap.Error(err))
			return nil, err
		}
	}

	return c.lastState, nil
}

func (c *HBCollector) collect(ctx context.Context) (*entities.State, error) {
	state := &entities.State{}
	state.Heartbeat = entities.Heartbeat{}

	c.logger.Debug("collecting heartbeat")
	c.logger.Debug("collecting IP")
	ip, err := utils.GetCurrentIP()
	if err != nil {
		c.logger.Error("failed to get current ip", zap.Error(err))
	}

	state.IPAddress = ip.String()
	state.Name = c.name
	state.Version = c.version

	c.logger.Debug("collecting prepared flag")
	state.Heartbeat.Prepared = utils.FileExists(entities.PreparedFlagPath)

	c.logger.Debug("collecting mariadb status")
	state.Heartbeat.MariadbStatus = c.mariadbManager.GetState()

	c.logger.Debug("collecting wsrep info")
	dsn := "root:root@unix(/run/mysqld/mysqld.sock)/"
	wsrepInfo, err := service.GetWsrepInfo(ctx, dsn)
	if err != nil {
		c.logger.Error("failed to get Wsrep info", zap.Error(err))
	}

	c.logger.Debug("got Wsrep info", zap.Any("wsrepInfo", wsrepInfo))

	state.WsrepInfo = entities.WsrepInfo{
		WsrepIncomingAddresses: wsrepInfo.WsrepIncomingAddresses,
		WsrepClusterSize:       wsrepInfo.WsrepClusterSize,
		WsrepClusterStatus:     wsrepInfo.WsrepClusterStatus,
	}

	c.mu.Lock()
	c.lastState, _ = state.Copy()
	c.mu.Unlock()

	return state, nil
}

func (c *HBCollector) send(ctx context.Context, state *entities.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	c.logger.Debug("sending heartbeat", zap.Any("heartbeat", string(data)))
	resp, err := c.httpClient.Post(fmt.Sprintf("http://%s/heartbeat", c.stateURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to send heartbeat", zap.Any("heartbeat", string(data)), zap.Int("status_code", resp.StatusCode))
	}

	return nil
}
