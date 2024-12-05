package main

import (
	"log/slog"
	"os"

	"github.com/jizhilong/branch-bot/config"
	"github.com/jizhilong/branch-bot/gitlab"
)

func main() {
	// setup slog to include source location
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	webhook, err := gitlab.NewWebhook(cfg.GitlabUrl, cfg.GitlabToken, cfg.RepoDirectory, cfg.BranchNamePrefix, cfg.ListenPort)
	if err != nil {
		slog.Error("Failed to create webhook", "error", err)
		os.Exit(1)
	}
	slog.Info("Starting webhook", "gitlab", cfg.GitlabUrl, "branchNamePrefix", cfg.BranchNamePrefix, "port", cfg.ListenPort)
	err = webhook.Start()
	if err != nil {
		slog.Error("Failed to start webhook", "error", err)
		os.Exit(1)
	}
}
