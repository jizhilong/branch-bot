package gitlab

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

// Client wraps gitlab.Client with additional functionality
type Client struct {
	*gitlab.Client
}

// NewClient creates a new GitLab client
func NewClient(baseURL, token string) (*Client, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}

	return &Client{
		Client: client,
	}, nil
}
