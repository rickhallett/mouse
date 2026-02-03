package gateway

import (
	"net/http"

	"mouse/internal/config"
	"mouse/internal/logging"
	"mouse/internal/telegram"
)

type Server struct {
	cfg    *config.Config
	logger *logging.Logger
	mux    *http.ServeMux
}

func NewServer(cfg *config.Config, logger *logging.Logger) *Server {
	mux := http.NewServeMux()
	server := &Server{cfg: cfg, logger: logger, mux: mux}

	mux.HandleFunc("/health", server.handleHealth)

	if cfg.Telegram.Enabled && cfg.Telegram.Webhook.Path != "" {
		tgHandler := telegram.NewHandler(telegram.Config{
			AllowFrom:      cfg.Telegram.AllowFrom,
			SecretToken:    cfg.Telegram.Webhook.Secret,
			RequireWebhook: cfg.Telegram.Webhook.Enabled,
		}, logging.New("telegram"))
		mux.Handle(cfg.Telegram.Webhook.Path, tgHandler)
	}

	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
