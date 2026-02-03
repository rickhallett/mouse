package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"mouse/internal/config"
	"mouse/internal/llm"
	"mouse/internal/logging"
	"mouse/internal/sessions"
	"mouse/internal/sqlite"
	"mouse/internal/telegram"
)

type Orchestrator struct {
	sessions *sessions.Store
	llm      llm.Client
	db       *sqlite.DB
	sender   *telegram.Sender
	logger   *logging.Logger
}

func New(cfg *config.Config, db *sqlite.DB, logger *logging.Logger) (*Orchestrator, error) {
	store, err := sessions.NewStore(cfg.Sessions.Dir)
	if err != nil {
		return nil, err
	}
	client, llmErr := llm.New(llm.Config{
		Provider:  cfg.LLM.Provider,
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		MaxTokens: cfg.LLM.MaxTokens,
	}, logging.New("llm"))
	if llmErr != nil && logger != nil {
		logger.Warn("llm client initialized with warnings", map[string]string{
			"error": llmErr.Error(),
		})
	}
	if db == nil {
		return nil, errors.New("sqlite db is required")
	}
	sender, err := telegram.NewSender(telegram.SenderConfig{
		BotToken:  cfg.Telegram.BotToken,
		AllowFrom: cfg.Telegram.AllowFrom,
	}, logging.New("telegram-outbound"))
	if err != nil {
		return nil, err
	}
	return &Orchestrator{
		sessions: store,
		llm:      client,
		db:       db,
		sender:   sender,
		logger:   logger,
	}, nil
}

func (o *Orchestrator) Process(ctx context.Context, update telegram.Update) (string, error) {
	if update.Message == nil {
		return "", errors.New("missing message")
	}
	if update.Message.Chat == nil {
		return "", errors.New("missing chat")
	}
	sessionID := strconv.FormatInt(update.Message.Chat.ID, 10)
	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return sessionID, errors.New("empty message")
	}
	if _, err := o.sessions.Append(sessionID, "user", text); err != nil {
		return sessionID, fmt.Errorf("append user message: %w", err)
	}
	if o.db != nil {
		if _, err := o.db.AppendSessionMessage(ctx, sessionID, "user", text); err != nil {
			return sessionID, fmt.Errorf("sqlite append user message: %w", err)
		}
	}
	response, err := o.llm.Complete(ctx, text)
	if err != nil {
		return sessionID, fmt.Errorf("llm completion: %w", err)
	}
	if _, err := o.sessions.Append(sessionID, "assistant", response); err != nil {
		return sessionID, fmt.Errorf("append assistant message: %w", err)
	}
	if o.db != nil {
		if _, err := o.db.AppendSessionMessage(ctx, sessionID, "assistant", response); err != nil {
			return sessionID, fmt.Errorf("sqlite append assistant message: %w", err)
		}
	}
	if o.sender != nil {
		if err := o.sender.SendMessage(ctx, update.Message.Chat.ID, update.Message.From, response); err != nil {
			return sessionID, fmt.Errorf("telegram send: %w", err)
		}
	}
	return sessionID, nil
}
