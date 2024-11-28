package main

import (
	"log/slog"
	"os"

	"github.com/jizhilong/light-merge/config"
	"github.com/jizhilong/light-merge/gitlab"
)

func main() {
	// Initialize logger with JSON handler
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	webhook, err := gitlab.NewWebhook(cfg.GitlabUrl, cfg.GitlabToken, cfg.RepoDirectory, cfg.ListenPort)
	if err != nil {
		slog.Error("Failed to create webhook", "error", err)
		os.Exit(1)
	}
	err = webhook.Start()
	if err != nil {
		slog.Error("Failed to start webhook", "error", err)
		os.Exit(1)
	}
}
