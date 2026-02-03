package cron

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mouse/internal/config"
	"mouse/internal/llm"
	"mouse/internal/logging"
	"mouse/internal/sessions"
	"mouse/internal/sqlite"
)

type Scheduler struct {
	cfg      config.CronConfig
	db       *sqlite.DB
	llm      llm.Client
	sessions *sessions.Store
	logger   *logging.Logger
	mu       sync.Mutex
	jobs     map[string]*job
}

type job struct {
	id       string
	schedule string
	session  string
	prompt   string
	next     time.Time
}

func New(cfg config.CronConfig, db *sqlite.DB, llmClient llm.Client, sessionsStore *sessions.Store, logger *logging.Logger) (*Scheduler, error) {
	if !cfg.Enabled {
		return nil, errors.New("cron: disabled")
	}
	if db == nil {
		return nil, errors.New("cron: db required")
	}
	if sessionsStore == nil {
		return nil, errors.New("cron: sessions required")
	}
	if llmClient == nil {
		return nil, errors.New("cron: llm required")
	}
	s := &Scheduler{
		cfg:      cfg,
		db:       db,
		llm:      llmClient,
		sessions: sessionsStore,
		logger:   logger,
		jobs:     make(map[string]*job),
	}
	if err := s.loadJobs(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
}

func (s *Scheduler) loadJobs() error {
	for _, jobCfg := range s.cfg.Jobs {
		parsed, err := parseSchedule(jobCfg.Schedule)
		if err != nil {
			return fmt.Errorf("cron: invalid schedule %s: %w", jobCfg.ID, err)
		}
		s.jobs[jobCfg.ID] = &job{
			id:       jobCfg.ID,
			schedule: jobCfg.Schedule,
			session:  jobCfg.Session,
			prompt:   jobCfg.Prompt,
			next:     parsed.next(time.Now().UTC()),
		}
		if err := s.db.UpsertCronJob(context.Background(), jobCfg, true); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) tick(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.jobs) == 0 {
		return
	}
	now := time.Now().UTC()
	for _, job := range s.jobs {
		if job.next.IsZero() || job.next.After(now) {
			continue
		}
		s.runJob(ctx, job)
		parsed, err := parseSchedule(job.schedule)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("cron schedule parse failed", map[string]string{
					"id":    job.id,
					"error": err.Error(),
				})
			}
			continue
		}
		job.next = parsed.next(now)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job *job) {
	prompt := strings.TrimSpace(job.prompt)
	if prompt == "" {
		return
	}
	if _, err := s.sessions.Append(job.session, "system", prompt); err != nil {
		if s.logger != nil {
			s.logger.Error("cron append prompt failed", map[string]string{
				"id":    job.id,
				"error": err.Error(),
			})
		}
		return
	}
	if _, err := s.db.AppendSessionMessage(ctx, job.session, "system", prompt); err != nil {
		if s.logger != nil {
			s.logger.Error("cron sqlite append prompt failed", map[string]string{
				"id":    job.id,
				"error": err.Error(),
			})
		}
		return
	}
	response, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("cron llm failed", map[string]string{
				"id":    job.id,
				"error": err.Error(),
			})
		}
		return
	}
	if _, err := s.sessions.Append(job.session, "assistant", response); err != nil {
		if s.logger != nil {
			s.logger.Error("cron append response failed", map[string]string{
				"id":    job.id,
				"error": err.Error(),
			})
		}
		return
	}
	if _, err := s.db.AppendSessionMessage(ctx, job.session, "assistant", response); err != nil {
		if s.logger != nil {
			s.logger.Error("cron sqlite append response failed", map[string]string{
				"id":    job.id,
				"error": err.Error(),
			})
		}
		return
	}
	if s.logger != nil {
		s.logger.Info("cron job executed", map[string]string{
			"id": job.id,
		})
	}
}

type schedule struct {
	minute int
	hour   int
}

func parseSchedule(expr string) (schedule, error) {
	parts := strings.Fields(expr)
	if len(parts) < 2 {
		return schedule{}, errors.New("cron: invalid expression")
	}
	minute, err := parseField(parts[0], 0, 59)
	if err != nil {
		return schedule{}, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return schedule{}, fmt.Errorf("hour: %w", err)
	}
	return schedule{minute: minute, hour: hour}, nil
}

func parseField(value string, min, max int) (int, error) {
	if value == "*" {
		return min, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func (s schedule) next(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), s.hour, s.minute, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
