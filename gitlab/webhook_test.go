package gitlab

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncRepo(t *testing.T) {
	// Setup
	glToken := os.Getenv("GITLAB_TOKEN")
	projectUrl := os.Getenv("GITLAB_REPO_URL")
	if glToken == "" || projectUrl == "" {
		t.Skip("environment GITLAB_TOKEN and GITLAB_REPO_URL are required")
		return
	}
	projectPath := "test-repo"
	repoDir, err := os.MkdirTemp("", "light-merge-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(repoDir)

	// Create a new Webhook instance
	webhook, err := NewWebhook(projectUrl, glToken, repoDir, "light-merges/", 8080)
	assert.NoError(t, err)

	t.Run("syncRepo success", func(t *testing.T) {
		// Test syncRepo function
		repo, err := webhook.syncRepo(projectPath, projectUrl)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Verify the repository was cloned correctly
		repoPath := fmt.Sprintf("%s/%s", repoDir, projectPath)
		_, err = os.Stat(repoPath)
		assert.NoError(t, err)

		// Verify the .git directory exists
		gitDirPath := fmt.Sprintf("%s/.git", repoPath)
		_, err = os.Stat(gitDirPath)
		assert.NoError(t, err)

		// Test syncRepo function
		repo, err = webhook.syncRepo(projectPath, projectUrl)
		assert.NoError(t, err)
		assert.NotNil(t, repo)
	})
	t.Run("syncRepo with invalid project URL", func(t *testing.T) {
		// Test syncRepo function
		_, err := webhook.syncRepo("invalid-repo", "http://localhost/invalid-repo.git")
		assert.Error(t, err)
		t.Log(err)
	})
}
