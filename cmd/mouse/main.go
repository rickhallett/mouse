package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"mouse/internal/config"
	"mouse/internal/gateway"
	"mouse/internal/logging"
	"mouse/internal/telegram"
)

func main() {
	var (
		configPath string
		addr       string
		checkOnly  bool
	)

	flag.StringVar(&configPath, "config", "./config/mouse.yaml", "path to config file")
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.BoolVar(&checkOnly, "check", false, "validate config and exit")
	flag.Parse()

	logger := logging.New("mouse")

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.EnsureRuntimeDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "runtime dirs error: %v\n", err)
		os.Exit(1)
	}

	if checkOnly {
		fmt.Println("config ok")
		return
	}

	if cfg.App.Workspace != "" {
		logPath := cfg.App.Workspace + "/logs/mouse.log"
		if err := logging.SetFile(logPath); err != nil {
			fmt.Fprintf(os.Stderr, "log setup error: %v\n", err)
			os.Exit(1)
		}
	}

	if cfg.Telegram.Enabled && cfg.Telegram.Webhook.Enabled {
		if err := telegram.SetWebhook(context.Background(), cfg.Telegram.BotToken, cfg.Telegram.Webhook.PublicURL, cfg.Telegram.Webhook.Path, cfg.Telegram.Webhook.Secret, logging.New("telegram-webhook")); err != nil {
			fmt.Fprintf(os.Stderr, "webhook error: %v\n", err)
			os.Exit(1)
		}
	}

	server, err := gateway.NewServer(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway error: %v\n", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("gateway starting", map[string]string{
		"addr": addr,
	})

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
