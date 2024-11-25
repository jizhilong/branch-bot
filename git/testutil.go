package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jizhilong/light-merge/models"
	"github.com/stretchr/testify/require"
)

// TestRepo extends Repo with testing utilities
type TestRepo struct {
	Repo
	t *testing.T
}

// NewTestRepo creates a new test repository
func NewTestRepo(t *testing.T) *TestRepo {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "light-merge-test-*")
	require.NoError(t, err)
	r := &TestRepo{Repo: Repo{path: tmpDir}, t: t}

	// Initialize git repo
	cmd := r.execCommand("git", "init")
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = r.execCommand("git", "config", "user.name", "test")
	require.NoError(t, cmd.Run())
	cmd = r.execCommand("git", "config", "user.email", "test@example.com")
	require.NoError(t, cmd.Run())

	// Create initial commit
	f, err := os.Create(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	_, err = f.WriteString("# Test Repo\n")
	require.NoError(t, err)
	f.Close()

	cmd = r.execCommand("git", "add", "README.md")
	require.NoError(t, cmd.Run())

	cmd = r.execCommand("git", "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return r
}

// CreateBranch creates a new branch with a test file
func (r *TestRepo) CreateBranch(base *models.GitRef, name, file, content string) *models.GitRef {
	// Reset to base commit
	cmd := r.execCommand("git", "reset", "--hard", base.Commit)
	require.NoError(r.t, cmd.Run())

	// Create a new branch
	cmd = r.execCommand("git", "checkout", "-b", name)
	require.NoError(r.t, cmd.Run())

	// Create a file
	f, err := os.Create(filepath.Join(r.path, file))
	require.NoError(r.t, err)
	_, err = f.WriteString(content)
	require.NoError(r.t, err)
	f.Close()

	// Add and commit
	cmd = r.execCommand("git", "add", file)
	require.NoError(r.t, cmd.Run())

	cmd = r.execCommand("git", "commit", "-m", "Add "+file)
	require.NoError(r.t, cmd.Run())

	// Get commit hash
	hash, err := r.RevParse("HEAD")
	require.NoError(r.t, err)

	return &models.GitRef{
		Name:   name,
		Commit: hash,
	}
}

func (r *TestRepo) UpdateBranch(name, file, content string) {
	cmd := r.execCommand("git", "checkout", name)
	require.NoError(r.t, cmd.Run())

	// Create a file
	f, err := os.Create(filepath.Join(r.path, file))
	require.NoError(r.t, err)
	_, err = f.WriteString(content)
	require.NoError(r.t, err)
	f.Close()

	// Add and commit
	cmd = r.execCommand("git", "add", file)
	require.NoError(r.t, cmd.Run())

	cmd = r.execCommand("git", "commit", "-m", "Update"+file)
	require.NoError(r.t, cmd.Run())

	// Get commit hash
	_, err = r.RevParse("HEAD")
	require.NoError(r.t, err)
}
