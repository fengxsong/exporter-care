package exporter

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

// Exporter todo
type Exporter struct {
	path      string
	port      int
	args, env []string
}

// New ..
func New(path string, port int, args, env []string) *Exporter {
	exp := &Exporter{
		path: path,
		port: port,
		env:  env,
	}
	flags := make([]string, 0)
	for i := range args {
		for _, sub := range strings.Split(args[i], " ") {
			trim := strings.TrimSpace(sub)
			if len(trim) > 0 {
				flags = append(flags, trim)
			}
		}
	}
	exp.args = flags
	return exp
}

// Build build command
func (e *Exporter) Build() (*exec.Cmd, int, error) {
	if path.IsAbs(e.path) {
		if _, err := os.Stat(e.path); err != nil {
			return nil, 0, err
		}
	} else {
		file, err := exec.LookPath(e.path)
		if err != nil {
			return nil, 0, err
		}
		e.path = file
	}
	var port int

	switch filepath.Base(e.path) {
	case "haproxy_exporter":
		port = 9101
	case "kafka_exporter":
		port = 9308
	case "mongodb_exporter":
		port = 9216
	case "mysqld_exporter":
		port = 9104
	case "node_exporter":
		port = 9100
	case "postgres_exporter":
		port = 9187
	case "rabbitmq_exporter":
		port = 9419
	case "redis_exporter":
		port = 9121
	case "php-fpm_exporter":
		port = 9253
	}
	if e.port == 0 {
		e.port = port
	}
	args, env := make([]string, 0), make([]string, 0)
	switch filepath.Base(e.path) {
	case "rabbitmq_exporter":
		env = append(env, fmt.Sprintf("PUBLISH_PORT=:%d", e.port))
	default:
		args = append(args, fmt.Sprintf("--web.listen-address=:%d", e.port))
	}
	cmd := exec.Command(e.path, append(e.args, args...)...)
	cmd.Env = append(os.Environ(), append(e.env, env...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setpgid = true
	return cmd, e.port, nil
}
