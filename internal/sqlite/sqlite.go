package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"mouse/internal/config"
)

type DB struct {
	db *sql.DB
}

type SessionMessage struct {
	ID        int64
	SessionID string
	Role      string
	Content   string
	CreatedAt string
}

type MemoryEntry struct {
	Key       string
	Content   string
	CreatedAt string
	UpdatedAt string
}

func Open(path string) (*DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite: path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite: create dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

func migrate(db *sql.DB) error {
	statements := []string{
		"PRAGMA journal_mode=WAL;",
		`CREATE TABLE IF NOT EXISTS session_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		"CREATE INDEX IF NOT EXISTS idx_session_messages_session_id ON session_messages(session_id, id);",
		`CREATE TABLE IF NOT EXISTS memory_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS cron_jobs (
			id TEXT PRIMARY KEY,
			schedule TEXT NOT NULL,
			session TEXT NOT NULL,
			prompt TEXT NOT NULL,
			enabled INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS index_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			last_indexed TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS index_entries (
			path TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			tokens TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("sqlite: migrate: %w", err)
		}
	}
	return nil
}

func (d *DB) AppendSessionMessage(ctx context.Context, sessionID, role, content string) (int64, error) {
	if d == nil || d.db == nil {
		return 0, errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return 0, errors.New("sqlite: session id is required")
	}
	if strings.TrimSpace(role) == "" {
		role = "user"
	}
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := d.db.ExecContext(ctx,
		"INSERT INTO session_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)",
		sessionID, strings.ToLower(role), content, timestamp,
	)
	if err != nil {
		return 0, fmt.Errorf("sqlite: insert session message: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (d *DB) ListSessionMessages(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("sqlite: db not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.QueryContext(ctx,
		"SELECT id, session_id, role, content, created_at FROM session_messages WHERE session_id = ? ORDER BY id DESC LIMIT ?",
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list session messages: %w", err)
	}
	defer rows.Close()

	var messages []SessionMessage
	for rows.Next() {
		var msg SessionMessage
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan session message: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate session messages: %w", err)
	}
	return messages, nil
}

func (d *DB) DeleteSession(ctx context.Context, sessionID string) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("sqlite: session id is required")
	}
	if _, err := d.db.ExecContext(ctx, "DELETE FROM session_messages WHERE session_id = ?", sessionID); err != nil {
		return fmt.Errorf("sqlite: delete session: %w", err)
	}
	return nil
}

func (d *DB) UpsertMemory(ctx context.Context, key, content string) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("sqlite: memory key is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO memory_entries (key, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = excluded.updated_at`,
		key, content, now, now,
	)
	if err != nil {
		return fmt.Errorf("sqlite: upsert memory: %w", err)
	}
	return nil
}

func (d *DB) GetMemory(ctx context.Context, key string) (*MemoryEntry, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("sqlite: memory key is required")
	}
	row := d.db.QueryRowContext(ctx,
		"SELECT key, content, created_at, updated_at FROM memory_entries WHERE key = ?",
		key,
	)
	var entry MemoryEntry
	if err := row.Scan(&entry.Key, &entry.Content, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("sqlite: get memory: %w", err)
	}
	return &entry, nil
}

func (d *DB) ListMemory(ctx context.Context, limit int) ([]MemoryEntry, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("sqlite: db not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.QueryContext(ctx,
		"SELECT key, content, created_at, updated_at FROM memory_entries ORDER BY updated_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list memory: %w", err)
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var entry MemoryEntry
		if err := rows.Scan(&entry.Key, &entry.Content, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan memory: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate memory: %w", err)
	}
	return entries, nil
}

func (d *DB) DeleteMemory(ctx context.Context, key string) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("sqlite: memory key is required")
	}
	if _, err := d.db.ExecContext(ctx, "DELETE FROM memory_entries WHERE key = ?", key); err != nil {
		return fmt.Errorf("sqlite: delete memory: %w", err)
	}
	return nil
}

type IndexEntry struct {
	Path        string
	Content     string
	Tokens      string
	ContentHash string
	UpdatedAt   string
}

func (d *DB) UpsertIndexEntry(ctx context.Context, path, content, tokens, hash string) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("sqlite: index path is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO index_entries (path, content, tokens, content_hash, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, tokens = excluded.tokens,
		 content_hash = excluded.content_hash, updated_at = excluded.updated_at`,
		path, content, tokens, hash, now,
	)
	if err != nil {
		return fmt.Errorf("sqlite: upsert index entry: %w", err)
	}
	return nil
}

func (d *DB) GetIndexHash(ctx context.Context, path string) (string, error) {
	if d == nil || d.db == nil {
		return "", errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("sqlite: index path is required")
	}
	row := d.db.QueryRowContext(ctx, "SELECT content_hash FROM index_entries WHERE path = ?", path)
	var hash string
	if err := row.Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("sqlite: get index hash: %w", err)
	}
	return hash, nil
}

func (d *DB) ListIndexEntries(ctx context.Context, limit int) ([]IndexEntry, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("sqlite: db not initialized")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := d.db.QueryContext(ctx,
		"SELECT path, content, tokens, content_hash, updated_at FROM index_entries ORDER BY updated_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list index entries: %w", err)
	}
	defer rows.Close()

	var entries []IndexEntry
	for rows.Next() {
		var entry IndexEntry
		if err := rows.Scan(&entry.Path, &entry.Content, &entry.Tokens, &entry.ContentHash, &entry.UpdatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan index entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate index entries: %w", err)
	}
	return entries, nil
}

func (d *DB) ListIndexPaths(ctx context.Context) ([]string, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("sqlite: db not initialized")
	}
	rows, err := d.db.QueryContext(ctx, "SELECT path FROM index_entries")
	if err != nil {
		return nil, fmt.Errorf("sqlite: list index paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("sqlite: scan index path: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate index paths: %w", err)
	}
	return paths, nil
}

func (d *DB) DeleteIndexEntry(ctx context.Context, path string) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("sqlite: index path is required")
	}
	if _, err := d.db.ExecContext(ctx, "DELETE FROM index_entries WHERE path = ?", path); err != nil {
		return fmt.Errorf("sqlite: delete index entry: %w", err)
	}
	return nil
}

func (d *DB) UpsertCronJob(ctx context.Context, job config.CronJob, enabled bool) error {
	if d == nil || d.db == nil {
		return errors.New("sqlite: db not initialized")
	}
	if strings.TrimSpace(job.ID) == "" {
		return errors.New("sqlite: cron job id required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	flag := 0
	if enabled {
		flag = 1
	}
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO cron_jobs (id, schedule, session, prompt, enabled, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET schedule = excluded.schedule, session = excluded.session,
		 prompt = excluded.prompt, enabled = excluded.enabled, updated_at = excluded.updated_at`,
		job.ID, job.Schedule, job.Session, job.Prompt, flag, now,
	)
	if err != nil {
		return fmt.Errorf("sqlite: upsert cron job: %w", err)
	}
	return nil
}
