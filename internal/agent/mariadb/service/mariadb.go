package service

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/vlourd/galera-k8s-operator/internal/agent/utils"
)

// ParseGraState parses galera grastate.dat and retrieves seqno and safeToBootstrap.
func ParseGraState(path string) (seqno, safeToBootstrap string, err error) {
	seqno = "-1"
	safeToBootstrap = "0"

	file, err := os.Open(path)
	if err != nil {
		return seqno, safeToBootstrap, fmt.Errorf("failed to open grastate file: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		const mustPathLen = 2
		parts := strings.Split(line, ":")
		if len(parts) != mustPathLen {
			return seqno, safeToBootstrap, fmt.Errorf("invalid string format: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "seqno":
			seqno = value
		case "safe_to_bootstrap":
			safeToBootstrap = value
		default:
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return seqno, safeToBootstrap, fmt.Errorf("failed to scan grastate file: %w", err)
	}

	return seqno, safeToBootstrap, nil
}

// Databases returns database list.
func Databases(ctx context.Context, dsn string) ([]string, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mariadb: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mariadb: %w", err)
	}

	query := `SHOW DATABASES`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	defer func() { _ = stmt.Close() }()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	databases := make([]string, 0)
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return databases, nil
			}

			return nil, fmt.Errorf("failed to scan databases: %w", err)
		}

		databases = append(databases, dbName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows: %w", err)
	}

	return databases, nil
}

// User contains mariadb user info.
type User struct {
	Username string
	Password string
	Host     string
}

// Users returns User list from mariadb.
func Users(ctx context.Context, dsn string) ([]User, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mariadb: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mariadb: %w", err)
	}

	query := `SELECT user, password, host FROM mysql.user`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	defer func() { _ = stmt.Close() }()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	users := make([]User, 0)
	for rows.Next() {
		user := User{}
		if err := rows.Scan(&user.Username, &user.Password, &user.Host); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows: %w", err)
	}

	return users, nil
}

// WsrepInfo contains wsrep info.
type WsrepInfo struct {
	WsrepClusterStatus     string
	WsrepClusterSize       int
	WsrepIncomingAddresses string
}

func GetWsrepInfo(ctx context.Context, dsn string) (WsrepInfo, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return WsrepInfo{}, fmt.Errorf("failed to connect to mariadb: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return WsrepInfo{}, fmt.Errorf("failed to ping mariadb: %w", err)
	}

	// Query to fetch all WSREP-related status variables
	// Using information_schema.GLOBAL_STATUS which is compatible with both MySQL and MariaDB
	query := `
SELECT VARIABLE_NAME, VARIABLE_VALUE 
FROM information_schema.GLOBAL_STATUS 
WHERE VARIABLE_NAME like 'wsrep%';`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return WsrepInfo{}, fmt.Errorf("failed to prepare query: %w", err)
	}

	defer func() { _ = stmt.Close() }()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return WsrepInfo{}, fmt.Errorf("failed to query wsrep: %w", err)
	}
	defer func() { _ = rows.Close() }()

	res := WsrepInfo{}
	rawValues := make(map[string]string)
	for rows.Next() {
		var (
			key string
			val string
		)
		err = rows.Scan(&key, &val)
		if err != nil {
			return WsrepInfo{}, fmt.Errorf("failed to scan key: %w", err)
		}

		// Store the raw key-value pair in the map
		rawValues[key] = val
	}

	res.WsrepClusterStatus = rawValues["WSREP_CLUSTER_STATUS"]
	res.WsrepClusterSize = utils.Atoi(rawValues["WSREP_CLUSTER_SIZE"])
	res.WsrepIncomingAddresses = rawValues["WSREP_INCOMING_ADDRESSES"]
	return res, nil
}
