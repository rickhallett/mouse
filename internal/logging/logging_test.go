package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetFileAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mouse.log")
	if err := SetFile(path); err != nil {
		t.Fatalf("set file: %v", err)
	}
	logger := New("test")
	logger.Info("hello", map[string]string{"k": "v"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "\"msg\":\"hello\"") {
		t.Fatalf("expected log line written")
	}
}
