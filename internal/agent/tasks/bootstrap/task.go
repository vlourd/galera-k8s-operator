package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/vlourd/galera-k8s-operator/internal/agent/utils"
)

const (
	//mariadbExecutable = `./waiter.sh`
	mariadbExecutable = "docker-entrypoint.sh"
	galeraConfPath    = "/etc/mysql/conf.d/galera.cnf"
)

var galeraConf = `
[mysqld]
wsrep_on                 = 1
wsrep_provider = /usr/lib/galera/libgalera_smm.so
binlog_format            = row
default_storage_engine   = InnoDB
innodb_autoinc_lock_mode = 2
wsrep_cluster_address = "gcomm://<node_hostnames>"
wsrep_cluster_name    = "galera-cluster"
wsrep_node_name       = "<node_name>"
wsrep_node_address    = "<node_hostname>"

wsrep_sst_method=rsync
wsrep_sst_auth=root:
`

func ConfigureGalera(logger *zap.Logger, pl *BootstrapPrimaryPayload) (bool, error) {
	ip, err := utils.GetCurrentIP()
	if err != nil {
		return false, errors.Wrap(err, "failed to get current ip")
	}

	hostnames := strings.Join(pl.ClusterNodes, ",")
	hostnames = fmt.Sprintf("%s,%s", hostnames, ip.String())
	cfg := strings.ReplaceAll(galeraConf, "<node_hostnames>", hostnames)
	cfg = strings.ReplaceAll(cfg, "<node_name>", ip.String())
	cfg = strings.ReplaceAll(cfg, "<node_hostname>", ip.String())

	oldHash, newHash := "", ""
	if utils.FileExists(galeraConfPath) {
		c, err := os.ReadFile(galeraConfPath)
		if err != nil {
			return false, errors.Wrap(err, "failed to read galera config")
		}
		hash := sha256.Sum256(c)
		oldHash = hex.EncodeToString(hash[:])
	}

	if oldHash != "" {
		hash := sha256.Sum256([]byte(cfg))
		newHash = hex.EncodeToString(hash[:])
		if newHash == oldHash {
			logger.Info("galera config is unchanged", zap.String("old_hash", oldHash))
			return false, nil
		}
	}

	logger.Info("rewriting galera config", zap.String("old_hash", oldHash))
	err = os.WriteFile("/etc/mysql/conf.d/galera.cnf", []byte(cfg), 0644)
	if err != nil {
		return false, errors.Wrap(err, "failed to rewrite galera configuration")
	}
	logger.Info("galera config has been rewritten", zap.String("new_hash", newHash))

	return true, nil
}
