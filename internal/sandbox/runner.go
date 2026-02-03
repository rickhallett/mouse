package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"mouse/internal/config"
)

type Runner struct {
	cfg config.SandboxConfig
}

type Result struct {
	Command  []string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

func New(cfg config.SandboxConfig) (*Runner, error) {
	if !cfg.Enabled {
		return nil, errors.New("sandbox: disabled")
	}
	if strings.TrimSpace(cfg.Docker.Image) == "" {
		return nil, errors.New("sandbox: docker image is required")
	}
	if len(cfg.Docker.Binds) == 0 {
		return nil, errors.New("sandbox: docker binds are required")
	}
	return &Runner{cfg: cfg}, nil
}

func (r *Runner) Run(ctx context.Context, command []string) (Result, error) {
	if r == nil {
		return Result{}, errors.New("sandbox: runner is nil")
	}
	if len(command) == 0 {
		return Result{}, errors.New("sandbox: command is required")
	}
	args := buildDockerArgs(r.cfg.Docker, command)
	execCmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	start := time.Now()
	err := execCmd.Run()
	result := Result{
		Command:  append([]string{"docker"}, args...),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
		Duration: time.Since(start),
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return result, err
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return result, nil
		}
		return result, fmt.Errorf("sandbox: command failed: %w", err)
	}
	return result, nil
}

func buildDockerArgs(cfg config.DockerConfig, command []string) []string {
	args := []string{"run", "--rm"}
	if cfg.ReadOnlyRoot {
		args = append(args, "--read-only")
	}
	network := strings.TrimSpace(cfg.Network)
	if network == "" {
		network = "none"
	}
	args = append(args, "--network", network)
	if workdir := strings.TrimSpace(cfg.Workdir); workdir != "" {
		args = append(args, "-w", workdir)
	}
	for _, bind := range cfg.Binds {
		trimmed := strings.TrimSpace(bind)
		if trimmed == "" {
			continue
		}
		args = append(args, "-v", trimmed)
	}
	for _, tmpfs := range cfg.Tmpfs {
		trimmed := strings.TrimSpace(tmpfs)
		if trimmed == "" {
			continue
		}
		args = append(args, "--tmpfs", trimmed)
	}
	args = append(args, cfg.Image)
	args = append(args, command...)
	return args
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
