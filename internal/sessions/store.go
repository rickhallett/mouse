package sessions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("sessions: dir is required")
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Append(sessionID, role, content string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "unknown"
	}
	if strings.TrimSpace(role) == "" {
		role = "user"
	}
	cleanID := sanitizeID(sessionID)
	path := filepath.Join(s.dir, cleanID+".md")
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return path, fmt.Errorf("sessions: create dir: %w", err)
	}
	newFile := false
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			newFile = true
		} else {
			return path, fmt.Errorf("sessions: stat: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return path, fmt.Errorf("sessions: open: %w", err)
	}
	defer file.Close()

	if newFile {
		header := fmt.Sprintf("# Session %s\n\n", sessionID)
		if _, err := file.WriteString(header); err != nil {
			return path, fmt.Errorf("sessions: write header: %w", err)
		}
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	entry := fmt.Sprintf("## %s %s\n\n%s\n\n", timestamp, strings.ToLower(role), strings.TrimSpace(content))
	if _, err := file.WriteString(entry); err != nil {
		return path, fmt.Errorf("sessions: write entry: %w", err)
	}
	return path, nil
}

func sanitizeID(id string) string {
	mapped := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '-' || r == '_' {
			return r
		}
		return '-'
	}, id)
	mapped = strings.Trim(mapped, "-")
	if mapped == "" {
		return "session"
	}
	if len(mapped) > 64 {
		return mapped[:64]
	}
	return mapped
}
