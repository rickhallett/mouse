package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	service string
}

var (
	fileMu   sync.Mutex
	filePath string
	file     *os.File
)

type entry struct {
	Timestamp string            `json:"ts"`
	Level     string            `json:"level"`
	Service   string            `json:"service"`
	Message   string            `json:"msg"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func New(service string) *Logger {
	return &Logger{service: service}
}

func SetFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("log mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("log open: %w", err)
	}
	fileMu.Lock()
	defer fileMu.Unlock()
	if file != nil {
		_ = file.Close()
	}
	file = f
	filePath = path
	return nil
}

func (l *Logger) Info(msg string, fields map[string]string) {
	l.write("info", msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]string) {
	l.write("warn", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]string) {
	l.write("error", msg, fields)
}

func (l *Logger) write(level, msg string, fields map[string]string) {
	e := entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Message:   msg,
		Fields:    fields,
	}
	b, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log marshal error: %v\n", err)
		return
	}
	line := string(b)
	fmt.Println(line)
	fileMu.Lock()
	defer fileMu.Unlock()
	if file != nil {
		_, _ = file.WriteString(line + "\n")
	}
}
