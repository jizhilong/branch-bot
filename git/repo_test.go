package git

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/jizhilong/light-merge/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	repo := NewTestRepo(t)
	// Get base commit
	baseHash, err := repo.RevParse("HEAD")
	require.NoError(t, err)
	base := &models.GitRef{Name: "main", Commit: baseHash}

	t.Run("single branch", func(t *testing.T) {
		ref := repo.CreateBranch(base, "feature1", "file1.txt", "feature1 content")
		result, fail := repo.Merge("Merge feature1 into main", base, ref)
		assert.NotNil(t, result)
		assert.Nil(t, fail)
	})

	t.Run("multiple branches without conflict", func(t *testing.T) {
		ref1 := repo.CreateBranch(base, "feature2", "file2.txt", "feature2 content")
		ref2 := repo.CreateBranch(base, "feature3", "file3.txt", "feature3 content")
		result, fail := repo.Merge("Merge feature2, feature3 into main", base, ref1, ref2)
		assert.NotNil(t, result)
		assert.Nil(t, fail)
	})

	t.Run("branches with conflict", func(t *testing.T) {
		ref1 := repo.CreateBranch(base, "conflict1", "conflict.txt", "content from branch1")
		ref2 := repo.CreateBranch(base, "conflict2", "conflict.txt", "content from branch2")
		result, fail := repo.Merge("Merge conflict1, conflict2 into main", base, ref1, ref2)
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		assert.NotEmpty(t, fail.FailedFiles)
		assert.Equal(t, "conflict.txt", fail.FailedFiles[0].Path)
	})

	t.Run("multiple branches with conflict", func(t *testing.T) {
		ref1 := repo.CreateBranch(base, "multi1", "multi.txt", "content from multi1")
		ref2 := repo.CreateBranch(base, "multi2", "other.txt", "content from multi2")
		ref3 := repo.CreateBranch(base, "multi3", "multi.txt", "content from multi3")
		result, fail := repo.Merge("Merge multi1, multi2, multi3 into main", base, ref1, ref2, ref3)
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		assert.Contains(t, fail.ConflictBranches, "multi1")
		assert.Equal(t, "multi3", fail.ConflictBranches[len(fail.ConflictBranches)-1])
	})
}

func TestGetCommitMessage(t *testing.T) {
	repo := NewTestRepo(t)

	// Get initial commit hash
	baseHash, err := repo.RevParse("HEAD")
	require.NoError(t, err)

	// Create a commit with specific message
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", "Test commit message\n\nDetailed description")
	cmd.Dir = repo.path
	require.NoError(t, cmd.Run())

	// Get new commit hash
	newHash, err := repo.RevParse("HEAD")
	require.NoError(t, err)

	tests := []struct {
		name    string
		commit  string
		want    string
		wantErr bool
	}{
		{
			name:    "get message from valid commit",
			commit:  newHash,
			want:    "Test commit message\n\nDetailed description\n",
			wantErr: false,
		},
		{
			name:    "get message from initial commit",
			commit:  baseHash,
			want:    "Initial commit\n",
			wantErr: false,
		},
		{
			name:    "invalid commit hash",
			commit:  "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetCommitMessage(tt.commit)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSyncRepo(t *testing.T) {
	projectPath := "test-repo"
	repoDir, err := os.MkdirTemp("", "light-merge-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(repoDir)
	repoPath := fmt.Sprintf("%s/%s", repoDir, projectPath)
	remoteUrl := "https://github.com/jizhilong/pywireguard.git"

	t.Run("syncRepo success", func(t *testing.T) {
		// Test syncRepo function
		repo, err := SyncRepo(repoPath, remoteUrl)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Verify the repository was cloned correctly
		_, err = os.Stat(repoPath)
		assert.NoError(t, err)

		// Verify the .git directory exists
		gitDirPath := fmt.Sprintf("%s/.git", repoPath)
		_, err = os.Stat(gitDirPath)
		assert.NoError(t, err)

		// Test syncRepo function
		repo, err = SyncRepo(repoPath, remoteUrl)
		assert.NoError(t, err)
		assert.NotNil(t, repo)
	})
	t.Run("syncRepo with invalid project URL", func(t *testing.T) {
		// Test syncRepo function
		_, err := SyncRepo("invalid-repo", "http://localhost/invalid-repo.git")
		assert.Error(t, err)
		t.Log(err)
	})
}
