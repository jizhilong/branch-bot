package config

import (
	"fmt"
	"os"
)

type Config struct {
	GitLab GitLabConfig
	Server ServerConfig
}

type GitLabConfig struct {
	URL   string
	Token string
}

type ServerConfig struct {
	Port int
}

func Load() (*Config, error) {
	gitlabURL := os.Getenv("LM_GITLAB_URL")
	if gitlabURL == "" {
		gitlabURL = "https://gitlab.com" // default value
	}

	gitlabToken := os.Getenv("LM_GITLAB_TOKEN")
	if gitlabToken == "" {
		return nil, fmt.Errorf("LM_GITLAB_TOKEN environment variable is required")
	}

	port := 8080 // default value
	// TODO: add port configuration if needed

	return &Config{
		GitLab: GitLabConfig{
			URL:   gitlabURL,
			Token: gitlabToken,
		},
		Server: ServerConfig{
			Port: port,
		},
	}, nil
}
