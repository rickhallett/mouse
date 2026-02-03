package sandbox

import (
	"context"
	"testing"

	"mouse/internal/config"
)

func TestBuildDockerArgs(t *testing.T) {
	cfg := config.DockerConfig{
		Image:        "mouse-sandbox:latest",
		Workdir:      "/workspace",
		Binds:        []string{"/host/runtime:/workspace:rw"},
		ReadOnlyRoot: true,
		Network:      "none",
		Tmpfs:        []string{"/tmp"},
	}
	args := buildDockerArgs(cfg, []string{"echo", "hi"})
	want := []string{
		"run", "--rm", "--read-only", "--network", "none", "-w", "/workspace",
		"-v", "/host/runtime:/workspace:rw", "--tmpfs", "/tmp", "mouse-sandbox:latest", "echo", "hi",
	}
	if len(args) != len(want) {
		t.Fatalf("unexpected args length: got %d want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg %d mismatch: got %q want %q", i, args[i], want[i])
		}
	}
}

func TestRunnerRequiresCommand(t *testing.T) {
	runner, err := New(config.SandboxConfig{
		Enabled: true,
		Docker: config.DockerConfig{
			Image: "mouse-sandbox:latest",
			Binds: []string{"/host/runtime:/workspace:rw"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected new error: %v", err)
	}
	_, err = runner.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for missing command")
	}
}
