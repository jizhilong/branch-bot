package main

import (
	"log/slog"
	"os"

	"github.com/jizhilong/light-merge/config"
	"github.com/jizhilong/light-merge/gitlab"
)

func main() {
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
