package utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	//mariadbExecutable = `./waiter.sh`
	mariadbExecutable = "docker-entrypoint.sh"
	galeraConfPath    = "/etc/mysql/conf.d/galera.cnf"
)

func GetCurrentIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File exists
	}
	if os.IsNotExist(err) {
		return false // File does not exist
	}
	return false // Error (e.g., permission issues)
}

// TouchFile creates an empty file and all necessary parent directories
// Similar to 'mkdir -p' + 'touch' in Unix
func TouchFile(path string) error {
	// Create all parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create or update the file
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	file.Close()

	// Update access and modification times (like standard touch)
	currentTime := time.Now()
	if err := os.Chtimes(path, currentTime, currentTime); err != nil {
		return fmt.Errorf("failed to update timestamps: %w", err)
	}

	return nil
}

func RunMariadb(ctx context.Context, logger *zap.Logger, args ...string) error {
	cmd := exec.CommandContext(ctx, mariadbExecutable)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "MYSQL_ROOT_PASSWORD=root")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to start mariadb")
	}

	return nil
}

// Atoi is a helper function that safely converts a string to an integer.
// Returns 0 if the string cannot be parsed.
func Atoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
