package entities

import (
	"encoding/json"
	"time"
)

type RunOptions struct {
	GaleraNewCluster bool
}

type RunOption func(*RunOptions)

func WithGaleraNewCluster() RunOption {
	return func(opt *RunOptions) {
		opt.GaleraNewCluster = true
	}
}

type State struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	// IPAddress contains the IP address of the data plane instance.
	IPAddress string `json:"ip_address"`

	// Device contains a list of block devices that will be mounted in the data plane instance.
	Devices []Device `json:"devices"`

	// Initialized contains the initialization status of the node.
	// If true, the node has been applied and the IPAddress and Devices were recorded for it.
	// The Initialized parameter allows you to skip the infrastructure preparation step.
	Initialized bool `json:"initialized"`

	// WsrepInfo contains some information about galera cluster state
	WsrepInfo WsrepInfo `json:"wsrep_info"`

	// Heartbeat contains the heartbeat data of a data plane instance.
	Heartbeat
}

type WsrepInfo struct {
	WsrepClusterStatus     string
	WsrepClusterSize       int
	WsrepIncomingAddresses string
}

type Device struct {
	Name       string `json:"name" yaml:"name"`
	Label      string `json:"label" yaml:"label"`
	MountPoint string `json:"mount_point" yaml:"mount_point"`
}

type Heartbeat struct {
	MariadbStatus      string `json:"mariadb_status"`
	WsrepStartPosition string `json:"wsrep_start_position"`
	SafeToBootstrap    string `json:"safe_to_bootstrap"`
	IsSuccess          bool   `json:"is_success"`
	Prepared           bool   `json:"prepared"`
	PreparedAt         string `json:"prepared_at"`
	Updated            string `json:"updated"`
}

// Copy returns a deep copy of the State struct using JSON marshaling
func (s *State) Copy() (*State, error) {
	if s == nil {
		return nil, nil
	}

	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var deepCopy State
	err = json.Unmarshal(data, &deepCopy)
	if err != nil {
		return nil, err
	}

	return &deepCopy, nil
}
