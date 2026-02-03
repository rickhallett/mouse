package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"mouse/internal/config"
	"mouse/internal/logging"
	"mouse/internal/sqlite"
)

type Indexer struct {
	cfg      config.IndexConfig
	db       *sqlite.DB
	logger   *logging.Logger
	interval time.Duration
	started  atomic.Bool
}

type Match struct {
	Path    string  `json:"path"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
}

func New(cfg config.IndexConfig, db *sqlite.DB, logger *logging.Logger) (*Indexer, error) {
	if db == nil {
		return nil, errors.New("indexer: db is required")
	}
	return &Indexer{cfg: cfg, db: db, logger: logger, interval: 10 * time.Second}, nil
}

func (i *Indexer) Start(ctx context.Context) {
	if i == nil {
		return
	}
	if !i.started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		_ = i.ScanOnce(ctx)
		ticker := time.NewTicker(i.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := i.ScanOnce(ctx); err != nil && i.logger != nil {
					i.logger.Warn("index scan failed", map[string]string{
						"error": err.Error(),
					})
				}
			}
		}
	}()
}

func (i *Indexer) ScanOnce(ctx context.Context) error {
	if i == nil {
		return errors.New("indexer: nil")
	}
	paths := i.cfg.Watch.Paths
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	for _, root := range paths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				return nil
			}
			seen[path] = struct{}{}
			return i.indexFile(ctx, path)
		})
		if err != nil {
			return fmt.Errorf("indexer: walk %s: %w", root, err)
		}
	}
	return i.removeMissing(ctx, seen)
}

func (i *Indexer) Search(ctx context.Context, query string, limit int) ([]Match, error) {
	if i == nil {
		return nil, errors.New("indexer: nil")
	}
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil, nil
	}
	entries, err := i.db.ListIndexEntries(ctx, 0)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	matches := make([]Match, 0, len(entries))
	for _, entry := range entries {
		entryTokens := strings.Fields(entry.Tokens)
		score := similarity(queryTokens, entryTokens)
		if score <= 0 {
			continue
		}
		matches = append(matches, Match{
			Path:    entry.Path,
			Score:   score,
			Snippet: snippet(entry.Content, 220),
		})
	}
	sort.Slice(matches, func(a, b int) bool {
		return matches[a].Score > matches[b].Score
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func (i *Indexer) indexFile(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("indexer: read %s: %w", path, err)
	}
	hash := hashContent(content)
	prevHash, err := i.db.GetIndexHash(ctx, path)
	if err != nil {
		return err
	}
	if prevHash == hash {
		return nil
	}
	tokens := tokenize(string(content))
	if err := i.db.UpsertIndexEntry(ctx, path, string(content), strings.Join(tokens, " "), hash); err != nil {
		return err
	}
	if i.logger != nil {
		i.logger.Info("indexed file", map[string]string{
			"path": path,
		})
	}
	return nil
}

func (i *Indexer) removeMissing(ctx context.Context, seen map[string]struct{}) error {
	paths, err := i.db.ListIndexPaths(ctx)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		if err := i.db.DeleteIndexEntry(ctx, path); err != nil {
			return err
		}
		if i.logger != nil {
			i.logger.Info("removed index entry", map[string]string{
				"path": path,
			})
		}
	}
	return nil
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return unique(tokens)
}

func unique(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	var out []string
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func similarity(queryTokens, entryTokens []string) float64 {
	if len(queryTokens) == 0 || len(entryTokens) == 0 {
		return 0
	}
	qset := make(map[string]struct{}, len(queryTokens))
	for _, token := range queryTokens {
		qset[token] = struct{}{}
	}
	intersection := 0
	union := len(qset)
	for _, token := range entryTokens {
		if _, ok := qset[token]; ok {
			intersection++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func snippet(content string, max int) string {
	content = strings.TrimSpace(content)
	if len(content) <= max {
		return content
	}
	return content[:max] + "..."
}

func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
