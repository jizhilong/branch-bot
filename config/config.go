package config

import (
	"fmt"
	"os"
)

type Config struct {
	GitlabUrl     string
	GitlabToken   string
	RepoDirectory string
	ListenPort    int
}

func Load() (*Config, error) {
	config := &Config{
		RepoDirectory: "/tmp/light-merge-builds",
		ListenPort:    8181,
		GitlabUrl:     os.Getenv("LM_GITLAB_URL"),
		GitlabToken:   os.Getenv("LM_GITLAB_TOKEN"),
	}
	var errors []string
	if config.GitlabUrl == "" {
		errors = append(errors, "LM_GITLAB_URL is required")
	}
	if config.GitlabToken == "" {
		errors = append(errors, "LM_GITLAB_TOKEN is required")
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", errors)
	}
	return config, nil
}
