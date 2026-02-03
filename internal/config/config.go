package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Telegram TelegramConfig `yaml:"telegram"`
	LLM      LLMConfig      `yaml:"llm"`
	Sessions SessionsConfig `yaml:"sessions"`
	Memory   MemoryConfig   `yaml:"memory"`
	Index    IndexConfig    `yaml:"index"`
	Sandbox  SandboxConfig  `yaml:"sandbox"`
	Cron     CronConfig     `yaml:"cron"`
}

type AppConfig struct {
	Name      string `yaml:"name"`
	Workspace string `yaml:"workspace"`
	Timezone  string `yaml:"timezone"`
}

type TelegramConfig struct {
	Enabled   bool            `yaml:"enabled"`
	Webhook   WebhookConfig   `yaml:"webhook"`
	BotToken  string          `yaml:"bot_token"`
	AllowFrom []string        `yaml:"allow_from"`
	Groups    TelegramGroups  `yaml:"groups"`
}

type WebhookConfig struct {
	Enabled   bool   `yaml:"enabled"`
	PublicURL string `yaml:"public_url"`
	Path      string `yaml:"path"`
	Secret    string `yaml:"secret"`
}

type TelegramGroups struct {
	Allow          []string `yaml:"allow"`
	RequireMention bool     `yaml:"require_mention"`
}

type LLMConfig struct {
	Provider  string `yaml:"provider"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

type SessionsConfig struct {
	Store              string `yaml:"store"`
	Dir                string `yaml:"dir"`
	MaxHistoryMessages int    `yaml:"max_history_messages"`
}

type MemoryConfig struct {
	Store    string `yaml:"store"`
	Dir      string `yaml:"dir"`
	AutoSync bool   `yaml:"auto_sync"`
}

type IndexConfig struct {
	SQLitePath string      `yaml:"sqlite_path"`
	Vector     VectorIndex `yaml:"vector"`
	Watch      WatchConfig `yaml:"watch"`
}

type VectorIndex struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
}

type WatchConfig struct {
	Paths []string `yaml:"paths"`
}

type SandboxConfig struct {
	Enabled bool           `yaml:"enabled"`
	Docker  DockerConfig   `yaml:"docker"`
	Tools   ToolPolicy     `yaml:"tools"`
}

type DockerConfig struct {
	Image        string   `yaml:"image"`
	Workdir      string   `yaml:"workdir"`
	Binds        []string `yaml:"binds"`
	ReadOnlyRoot bool     `yaml:"read_only_root"`
	Network      string   `yaml:"network"`
	Tmpfs        []string `yaml:"tmpfs"`
}

type ToolPolicy struct {
	Allow []string `yaml:"allow"`
	Deny  []string `yaml:"deny"`
}

type CronConfig struct {
	Enabled bool      `yaml:"enabled"`
	Jobs    []CronJob `yaml:"jobs"`
}

type CronJob struct {
	ID       string `yaml:"id"`
	Schedule string `yaml:"schedule"`
	Session  string `yaml:"session"`
	Prompt   string `yaml:"prompt"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.expandEnv()
	cfg.expandWorkspaceRefs()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) expandEnv() {
	c.Telegram.BotToken = expandEnvValue(c.Telegram.BotToken)
	c.Telegram.Webhook.Secret = expandEnvValue(c.Telegram.Webhook.Secret)
	c.LLM.APIKey = expandEnvValue(c.LLM.APIKey)
}

func expandEnvValue(value string) string {
	const prefix = "env:"
	if !strings.HasPrefix(value, prefix) {
		return value
	}
	key := strings.TrimSpace(strings.TrimPrefix(value, prefix))
	if key == "" {
		return ""
	}
	return os.Getenv(key)
}

func (c *Config) expandWorkspaceRefs() {
	workspace := c.App.Workspace
	if workspace == "" {
		return
	}
	c.Sessions.Dir = expandWorkspace(c.Sessions.Dir, workspace)
	c.Memory.Dir = expandWorkspace(c.Memory.Dir, workspace)
	c.Index.SQLitePath = expandWorkspace(c.Index.SQLitePath, workspace)
	for i := range c.Index.Watch.Paths {
		c.Index.Watch.Paths[i] = expandWorkspace(c.Index.Watch.Paths[i], workspace)
	}
	for i := range c.Sandbox.Docker.Binds {
		c.Sandbox.Docker.Binds[i] = expandWorkspace(c.Sandbox.Docker.Binds[i], workspace)
	}
}

func expandWorkspace(value, workspace string) string {
	const token = "${app.workspace}"
	return strings.ReplaceAll(value, token, workspace)
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		return errors.New("config: app.name is required")
	}
	if c.App.Workspace == "" {
		return errors.New("config: app.workspace is required")
	}
	if c.Telegram.Enabled {
		if c.Telegram.BotToken == "" {
			return errors.New("config: telegram.bot_token is required when telegram.enabled is true")
		}
		if len(c.Telegram.AllowFrom) == 0 {
			return errors.New("config: telegram.allow_from must include at least one sender")
		}
		if c.Telegram.Webhook.Enabled {
			if c.Telegram.Webhook.Path == "" {
				return errors.New("config: telegram.webhook.path is required when webhook is enabled")
			}
			if c.Telegram.Webhook.PublicURL == "" {
				return errors.New("config: telegram.webhook.public_url is required when webhook is enabled")
			}
		}
	}
	if c.Sessions.Store != "markdown" {
		return fmt.Errorf("config: sessions.store must be markdown, got %q", c.Sessions.Store)
	}
	if c.Memory.Store != "markdown" {
		return fmt.Errorf("config: memory.store must be markdown, got %q", c.Memory.Store)
	}
	if c.Sandbox.Enabled {
		if c.Sandbox.Docker.Image == "" {
			return errors.New("config: sandbox.docker.image is required when sandbox is enabled")
		}
		if len(c.Sandbox.Docker.Binds) == 0 {
			return errors.New("config: sandbox.docker.binds must include workspace bind")
		}
	}
	return nil
}

func (c *Config) EnsureRuntimeDirs() error {
	dirs := []string{
		c.App.Workspace,
		c.Sessions.Dir,
		c.Memory.Dir,
		filepath.Dir(c.Index.SQLitePath),
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}
