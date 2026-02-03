package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mouse/internal/logging"
)

const telegramAPIBase = "https://api.telegram.org"

type SenderConfig struct {
	BotToken  string
	AllowFrom []string
}

type Sender struct {
	botToken   string
	allowFrom  []string
	httpClient *http.Client
	logger     *logging.Logger
}

func NewSender(cfg SenderConfig, logger *logging.Logger) (*Sender, error) {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, errors.New("telegram: bot token is required")
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return &Sender{
		botToken:   cfg.BotToken,
		allowFrom:  cfg.AllowFrom,
		httpClient: client,
		logger:     logger,
	}, nil
}

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func (s *Sender) SendMessage(ctx context.Context, chatID int64, user *User, text string) error {
	if !isAllowedUser(s.allowFrom, user) {
		return errors.New("telegram: user not allowed")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("telegram: message text is empty")
	}
	if chatID == 0 {
		return errors.New("telegram: chat id is required")
	}

	payload := sendMessageRequest{ChatID: chatID, Text: text}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIBase, s.botToken)
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		status, respBody, reqErr := s.doRequest(ctx, url, body)
		if reqErr == nil && status >= 200 && status < 300 {
			return nil
		}
		if reqErr != nil {
			lastErr = reqErr
			if s.logger != nil {
				s.logger.Error("telegram send failed", map[string]string{
					"error":   reqErr.Error(),
					"attempt": strconv.Itoa(attempt),
				})
			}
		} else {
			trimmed := strings.TrimSpace(string(respBody))
			lastErr = fmt.Errorf("telegram: http %d: %s", status, trimmed)
			if s.logger != nil {
				s.logger.Error("telegram send failed", map[string]string{
					"status":  strconv.Itoa(status),
					"body":    trimmed,
					"attempt": strconv.Itoa(attempt),
				})
			}
		}
		if attempt == 1 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return lastErr
}

func (s *Sender) doRequest(ctx context.Context, url string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("telegram: read response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}
