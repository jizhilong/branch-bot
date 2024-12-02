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
	r.mustExec("git", "init")

	// Configure git
	r.mustExec("git", "config", "user.name", "test")
	r.mustExec("git", "config", "user.email", "test@example.com")

	// Create initial commit
	f, err := os.Create(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	_, err = f.WriteString("# Test Repo\n")
	require.NoError(t, err)
	f.Close()

	r.mustExec("git", "add", "README.md")

	r.mustExec("git", "commit", "-m", "Initial commit")

	// make sure the initial branch name is main
	r.mustExec("git", "branch", "-m", "main")

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return r
}

// CreateBranch creates a new branch with a test file
func (r *TestRepo) CreateBranch(base *models.GitRef, name, file, content string) *models.GitRef {
	// Create a new branch
	r.mustExec("git", "checkout", base.Commit, "-b", name)

	// Create a file
	f, err := os.Create(filepath.Join(r.path, file))
	require.NoError(r.t, err)
	_, err = f.WriteString(content)
	require.NoError(r.t, err)
	f.Close()

	// Add and commit
	r.mustExec("git", "add", file)

	r.mustExec("git", "commit", "-m", "Add "+file)

	// Get commit hash
	hash, err := r.RevParse("HEAD")
	require.NoError(r.t, err)

	return &models.GitRef{
		Name:   name,
		Commit: hash,
	}
}

func (r *TestRepo) UpdateBranch(name, file, content string) *models.GitRef {
	r.mustExec("git", "checkout", name)

	// Create a file
	f, err := os.Create(filepath.Join(r.path, file))
	require.NoError(r.t, err)
	_, err = f.WriteString(content)
	require.NoError(r.t, err)
	f.Close()

	// Add and commit
	r.mustExec("git", "add", file)

	r.mustExec("git", "commit", "-m", "Update"+file)

	// Get commit hash
	commit, err := r.RevParse("HEAD")
	require.NoError(r.t, err)
	return &models.GitRef{
		Name:   name,
		Commit: commit,
	}
}

func (r *TestRepo) mustExec(name string, args ...string) {
	_, err := r.execCommand(name, args...)
	if err != nil {
		r.t.Fatal(err)
	}
}
