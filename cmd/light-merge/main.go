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

	// Initialize GitLab client
	client, err := gitlab.NewClient(cfg.GitLab.URL, cfg.GitLab.Token)
	if err != nil {
		slog.Error("Failed to create GitLab client", "error", err)
		os.Exit(1)
	}

	slog.Info("Light-merge starting up...",
		"gitlab_url", cfg.GitLab.URL,
		"server_port", cfg.Server.Port,
		"client", client)
	// TODO: Start HTTP server
}
