package gateway

import (
	"context"
	"net/http"

	"mouse/internal/approvals"
	"mouse/internal/config"
	"mouse/internal/cron"
	"mouse/internal/indexer"
	"mouse/internal/llm"
	"mouse/internal/logging"
	"mouse/internal/orchestrator"
	"mouse/internal/sandbox"
	"mouse/internal/sessions"
	"mouse/internal/sqlite"
	"mouse/internal/telegram"
	"mouse/internal/tools"
)

type Server struct {
	cfg    *config.Config
	logger *logging.Logger
	mux    *http.ServeMux
	db     *sqlite.DB
}

func NewServer(cfg *config.Config, logger *logging.Logger) (*Server, error) {
	mux := http.NewServeMux()
	db, err := sqlite.Open(cfg.Index.SQLitePath)
	if err != nil {
		logger.Error("sqlite init failed", map[string]string{
			"error": err.Error(),
		})
		return nil, err
	}
	server := &Server{cfg: cfg, logger: logger, mux: mux, db: db}

	mux.HandleFunc("/health", server.handleHealth)
	mux.Handle("/approvals/submit", approvals.NewHandler(logging.New("approvals")))

	if cfg.Telegram.Enabled && cfg.Telegram.Webhook.Path != "" {
		orch, err := orchestrator.New(cfg, db, logging.New("orchestrator"))
		if err != nil {
			logger.Error("orchestrator init failed", map[string]string{
				"error": err.Error(),
			})
			return nil, err
		}
		tgHandler := telegram.NewHandler(telegram.Config{
			AllowFrom:      cfg.Telegram.AllowFrom,
			SecretToken:    cfg.Telegram.Webhook.Secret,
			RequireWebhook: cfg.Telegram.Webhook.Enabled,
		}, logging.New("telegram"), orch)
		mux.Handle(cfg.Telegram.Webhook.Path, tgHandler)
	}

	if cfg.Sandbox.Enabled {
		runner, err := sandbox.New(cfg.Sandbox)
		if err != nil {
			logger.Error("sandbox init failed", map[string]string{
				"error": err.Error(),
			})
			return nil, err
		}
		policy := tools.NewPolicy(cfg.Sandbox.Tools.Allow, cfg.Sandbox.Tools.Deny)
		toolHandler := tools.NewHandler(policy, runner, logging.New("tools"))
		mux.Handle("/tools/run", toolHandler)
	}

	cronClient, cronErr := llm.New(llm.Config{
		Provider:  cfg.LLM.Provider,
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		MaxTokens: cfg.LLM.MaxTokens,
	}, logging.New("cron-llm"))
	if cronErr != nil {
		logger.Warn("cron llm init failed", map[string]string{
			"error": cronErr.Error(),
		})
	}
	if cfg.Cron.Enabled && cronClient != nil {
		sessionStore, err := sessions.NewStore(cfg.Sessions.Dir)
		if err != nil {
			logger.Error("cron session store init failed", map[string]string{
				"error": err.Error(),
			})
			return nil, err
		}
		scheduler, err := cron.New(cfg.Cron, db, cronClient, sessionStore, logging.New("cron"))
		if err != nil {
			logger.Error("cron init failed", map[string]string{
				"error": err.Error(),
			})
			return nil, err
		}
		scheduler.Start(context.Background())
	}

	if len(cfg.Index.Watch.Paths) > 0 {
		idx, err := indexer.New(cfg.Index, db, logging.New("indexer"))
		if err != nil {
			logger.Error("indexer init failed", map[string]string{
				"error": err.Error(),
			})
			return nil, err
		}
		idx.Start(context.Background())
		mux.Handle("/index/search", indexer.NewHandler(idx, logging.New("indexer-http")))
		mux.Handle("/index/reindex", indexer.NewReindexHandler(idx, logging.New("indexer-http")))
	}

	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
