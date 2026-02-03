package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Logger struct {
	service string
}

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
	fmt.Println(string(b))
}
