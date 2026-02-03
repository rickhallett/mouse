package llm

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

const (
	anthropicVersion = "2023-06-01"
	anthropicURL     = "https://api.anthropic.com/v1/messages"
)

type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type Config struct {
	Provider  string
	APIKey    string
	Model     string
	MaxTokens int
}

func New(cfg Config, logger *logging.Logger) (Client, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case "claude", "anthropic":
		if strings.TrimSpace(cfg.APIKey) == "" {
			return &Noop{reason: "missing api key"}, errors.New("llm: missing api key")
		}
		if strings.TrimSpace(cfg.Model) == "" {
			return &Noop{reason: "missing model"}, errors.New("llm: missing model")
		}
		maxTokens := cfg.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 1024
		}
		return &anthropicClient{
			apiKey:     cfg.APIKey,
			model:      cfg.Model,
			maxTokens:  maxTokens,
			baseURL:    anthropicURL,
			version:    anthropicVersion,
			httpClient: &http.Client{Timeout: 30 * time.Second},
			logger:     logger,
		}, nil
	default:
		if provider == "" {
			provider = "unknown"
		}
		return &Noop{reason: "unsupported provider: " + provider}, fmt.Errorf("llm: unsupported provider %q", provider)
	}
}

type Noop struct {
	reason string
}

func (n *Noop) Complete(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("llm disabled: %s", n.reason)
}

type anthropicClient struct {
	apiKey     string
	model      string
	maxTokens  int
	baseURL    string
	version    string
	httpClient *http.Client
	logger     *logging.Logger
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messagesResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

func (c *anthropicClient) Complete(ctx context.Context, prompt string) (string, error) {
	payload := messagesRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("llm: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llm: request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm: http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}
	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text, nil
		}
	}
	if c.logger != nil {
		c.logger.Warn("llm response contained no text", map[string]string{
			"stop_reason": parsed.StopReason,
		})
	}
	return "", errors.New("llm: empty response")
}
