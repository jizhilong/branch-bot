package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jizhilong/light-merge/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepo(t *testing.T) (*Repo, func()) {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "light-merge-test-*")
	require.NoError(t, err)

	// Create repo instance first so we can use execCommand
	repo, err := New(tmpDir)
	require.NoError(t, err)

	// Initialize git repo
	require.NoError(t, repo.execCommand("git", "init").Run())

	// Configure git
	require.NoError(t, repo.execCommand("git", "config", "user.name", "test").Run())
	require.NoError(t, repo.execCommand("git", "config", "user.email", "test@example.com").Run())

	// Create initial commit
	f, err := os.Create(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	_, err = f.WriteString("# Test Repo\n")
	require.NoError(t, err)
	f.Close()

	require.NoError(t, repo.execCommand("git", "add", "README.md").Run())
	require.NoError(t, repo.execCommand("git", "commit", "-m", "Initial commit").Run())

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func createBranch(t *testing.T, repo *Repo, base *models.GitRef, name, file, content string) *models.GitRef {
	// Reset to base commit
	require.NoError(t, repo.execCommand("git", "reset", "--hard", base.Commit).Run())

	// Create a new branch
	require.NoError(t, repo.execCommand("git", "checkout", "-b", name).Run())

	// Create a file
	f, err := os.Create(filepath.Join(repo.path, file))
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	f.Close()

	// Add and commit
	require.NoError(t, repo.execCommand("git", "add", file).Run())
	require.NoError(t, repo.execCommand("git", "commit", "-m", "Add "+file).Run())

	// Get commit hash
	hash, err := repo.RevParse("HEAD")
	require.NoError(t, err)

	return &models.GitRef{
		Name:   name,
		Commit: hash,
	}
}

func TestMerge(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get base commit
	baseHash, err := repo.RevParse("HEAD")
	require.NoError(t, err)
	base := &models.GitRef{Name: "main", Commit: baseHash}

	t.Run("single branch", func(t *testing.T) {
		ref := createBranch(t, repo, base, "feature1", "file1.txt", "feature1 content")
		result, fail := repo.Merge(base, ref)
		assert.NotNil(t, result)
		assert.Nil(t, fail)
	})

	t.Run("multiple branches without conflict", func(t *testing.T) {
		ref1 := createBranch(t, repo, base, "feature2", "file2.txt", "feature2 content")
		ref2 := createBranch(t, repo, base, "feature3", "file3.txt", "feature3 content")
		result, fail := repo.Merge(base, ref1, ref2)
		assert.NotNil(t, result)
		assert.Nil(t, fail)
	})

	t.Run("branches with conflict", func(t *testing.T) {
		ref1 := createBranch(t, repo, base, "conflict1", "conflict.txt", "content from branch1")
		ref2 := createBranch(t, repo, base, "conflict2", "conflict.txt", "content from branch2")
		result, fail := repo.Merge(base, ref1, ref2)
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		assert.NotEmpty(t, fail.FailedFiles)
		assert.Equal(t, "conflict.txt", fail.FailedFiles[0].Path)
	})

	t.Run("multiple branches with conflict", func(t *testing.T) {
		ref1 := createBranch(t, repo, base, "multi1", "multi.txt", "content from multi1")
		ref2 := createBranch(t, repo, base, "multi2", "other.txt", "content from multi2")
		ref3 := createBranch(t, repo, base, "multi3", "multi.txt", "content from multi3")
		result, fail := repo.Merge(base, ref1, ref2, ref3)
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		assert.Contains(t, fail.ConflictBranches, "multi1")
		assert.Equal(t, "multi3", fail.ConflictBranches[len(fail.ConflictBranches)-1])
	})
}
