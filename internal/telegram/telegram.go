package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"mouse/internal/logging"
)

type Config struct {
	AllowFrom      []string
	SecretToken    string
	RequireWebhook bool
}

type Handler struct {
	cfg    Config
	logger *logging.Logger
	proc   Processor
}

type Processor interface {
	Process(ctx context.Context, update Update) (string, error)
}

func NewHandler(cfg Config, logger *logging.Logger, proc Processor) *Handler {
	return &Handler{cfg: cfg, logger: logger, proc: proc}
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.cfg.SecretToken != "" {
		secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if secret != h.cfg.SecretToken {
			h.logger.Warn("telegram webhook secret mismatch", map[string]string{
				"remote": r.RemoteAddr,
			})
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("telegram read body failed", map[string]string{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		h.logger.Warn("telegram invalid json", map[string]string{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	allowed := isAllowedUpdate(h.cfg.AllowFrom, update)
	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if h.proc == nil {
		h.logger.Error("telegram processor not configured", nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionID, err := h.proc.Process(r.Context(), update)
	if err != nil {
		fields := map[string]string{"error": err.Error()}
		if sessionID != "" {
			fields["session_id"] = sessionID
		}
		h.logger.Error("telegram processing failed", fields)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fields := map[string]string{
		"update_id": strconv.FormatInt(update.UpdateID, 10),
	}
	if update.Message != nil && update.Message.From != nil {
		fields["from_id"] = strconv.FormatInt(update.Message.From.ID, 10)
		fields["from_user"] = update.Message.From.Username
	}
	if update.Message != nil && update.Message.Chat != nil {
		fields["chat_id"] = strconv.FormatInt(update.Message.Chat.ID, 10)
		fields["chat_type"] = update.Message.Chat.Type
	}
	h.logger.Info("telegram update received", fields)

	w.WriteHeader(http.StatusOK)
}
