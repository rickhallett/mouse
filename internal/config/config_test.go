package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleConfig = `app:
  name: mouse
  workspace: ./runtime
  timezone: UTC

telegram:
  enabled: true
  webhook:
    enabled: true
    public_url: "https://example.test/telegram-webhook"
    path: "/telegram-webhook"
    secret: "env:TEST_TG_SECRET"
  bot_token: "env:TEST_TG_TOKEN"
  allow_from:
    - "123456789"
  groups:
    allow: []
    require_mention: true

llm:
  provider: claude
  api_key: "env:TEST_ANTHROPIC"
  model: "claude-opus-4-5"
  max_tokens: 4096

sessions:
  store: markdown
  dir: "${app.workspace}/sessions"
  max_history_messages: 50

memory:
  store: markdown
  dir: "${app.workspace}/memory"
  auto_sync: true

index:
  sqlite_path: "${app.workspace}/sqlite/mouse.db"
  vector:
    enabled: true
    provider: "sqlite-vss"
  watch:
    paths:
      - "${app.workspace}/memory"
      - "${app.workspace}/sessions"
      - "${app.workspace}/notes"

sandbox:
  enabled: true
  docker:
    image: "mouse-sandbox:latest"
    workdir: "/workspace"
    binds:
      - "${app.workspace}:/workspace:rw"
    read_only_root: true
    network: "none"
    tmpfs:
      - "/tmp"
  tools:
    allow:
      - read
      - write
    deny:
      - exec

cron:
  enabled: true
  jobs: []
`

func TestLoadConfig(t *testing.T) {
	t.Setenv("TEST_TG_SECRET", "secret")
	t.Setenv("TEST_TG_TOKEN", "token")
	t.Setenv("TEST_ANTHROPIC", "key")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(sampleConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Telegram.BotToken != "token" {
		t.Fatalf("expected telegram token to resolve from env")
	}
	if cfg.Telegram.Webhook.Secret != "secret" {
		t.Fatalf("expected webhook secret to resolve from env")
	}
	if cfg.LLM.APIKey != "key" {
		t.Fatalf("expected llm api key to resolve from env")
	}
	if cfg.Sessions.Dir != "./runtime/sessions" {
		t.Fatalf("expected workspace expansion for sessions.dir, got %q", cfg.Sessions.Dir)
	}
}
