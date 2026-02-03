package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mouse/internal/logging"
)

func SetWebhook(ctx context.Context, token, publicURL, path, secret string, logger *logging.Logger) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("telegram: bot token required")
	}
	if strings.TrimSpace(publicURL) == "" {
		return errors.New("telegram: public url required")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("telegram: webhook path required")
	}
	url := strings.TrimRight(publicURL, "/") + path
	payload := map[string]any{
		"url": url,
	}
	if strings.TrimSpace(secret) != "" {
		payload["secret_token"] = secret
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal webhook: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	endpoint := fmt.Sprintf("%s/bot%s/setWebhook", telegramAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: request webhook: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: webhook call: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("telegram: webhook response read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: webhook status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if logger != nil {
		logger.Info("telegram webhook set", map[string]string{"url": url})
	}
	return nil
}
