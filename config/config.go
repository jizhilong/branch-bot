package config

import (
	"fmt"
	"os"
)

type Config struct {
	GitlabUrl   string
	GitlabToken string
	// RepoDirectory is a directory where repositories will be cloned
	RepoDirectory string
	// BranchNamePrefix output branch will be named as BranchNamePrefix + issue iid
	BranchNamePrefix string
	ListenPort       int
}

func Load() (*Config, error) {
	config := &Config{
		GitlabUrl:        os.Getenv("BB_GITLAB_URL"),
		GitlabToken:      os.Getenv("BB_GITLAB_TOKEN"),
		RepoDirectory:    os.Getenv("BB_REPO_DIRECTORY"),
		BranchNamePrefix: os.Getenv("BB_BRANCH_NAME_PREFIX"),
		ListenPort:       8181,
	}
	var errors []string
	if config.GitlabUrl == "" {
		errors = append(errors, "BB_GITLAB_URL is required")
	}
	if config.GitlabToken == "" {
		errors = append(errors, "BB_GITLAB_TOKEN is required")
	}
	if config.RepoDirectory == "" {
		config.RepoDirectory = "/tmp/bb-builds"
	}
	if config.BranchNamePrefix == "" {
		config.BranchNamePrefix = "bb-branches/"
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", errors)
	}
	return config, nil
}
