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
		GitlabUrl:        os.Getenv("LM_GITLAB_URL"),
		GitlabToken:      os.Getenv("LM_GITLAB_TOKEN"),
		RepoDirectory:    os.Getenv("LM_REPO_DIRECTORY"),
		BranchNamePrefix: os.Getenv("LM_BRANCH_NAME_PREFIX"),
		ListenPort:       8181,
	}
	var errors []string
	if config.GitlabUrl == "" {
		errors = append(errors, "LM_GITLAB_URL is required")
	}
	if config.GitlabToken == "" {
		errors = append(errors, "LM_GITLAB_TOKEN is required")
	}
	if config.RepoDirectory == "" {
		config.RepoDirectory = "/tmp/light-merge-builds"
	}
	if config.BranchNamePrefix == "" {
		config.BranchNamePrefix = "light-merges/"
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", errors)
	}
	return config, nil
}

func canRenderWithInteger(template string) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in canRenderWithInteger:", r)
		}
	}()

	// Try to format the string with an integer
	_ = fmt.Sprintf(template, 1)
	return true
}
