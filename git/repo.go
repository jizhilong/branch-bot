package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jizhilong/light-merge/models"
)

// Repo represents a Git repository
type Repo struct {
	path string // absolute path to the repository
}

// SyncRepo ensures the repository exists and is up-to-date with the remote
func SyncRepo(repoPath, remoteUrl string) (*Repo, error) {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create repo directory: %w", err)
		}
	}
	gitDirPath := fmt.Sprintf("%s/.git", repoPath)
	var repo *Repo
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		// clone from remote
		repo, err = Clone(remoteUrl, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
		if err = repo.Config("user.name", "light-merge"); err != nil {
			return nil, fmt.Errorf("failed to set user name: %w", err)
		}
		if err = repo.Config("user.email", "operator@light-merge.localhost"); err != nil {
			return nil, fmt.Errorf("failed to set user email: %w", err)
		}
	} else {
		repo, err = New(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository: %w", err)
		}
		err = repo.RefreshRemote()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh remote: %w", err)
		}
	}
	return repo, nil
}

func Clone(url, path string) (*Repo, error) {
	cmd := exec.Command("git", "clone", url, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %s: %w", output, err)
	}
	return New(path)
}

// New creates a new Repo instance
func New(path string) (*Repo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if absPath existed
	if fileInfo, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	} else if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	return &Repo{path: absPath}, nil
}

// execCommand creates and executes a git command in the repo directory
func (r *Repo) execCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = r.path
	return cmd
}

// RevParse returns the commit hash for the given revision
func (r *Repo) RevParse(rev string) (string, error) {
	output, err := r.execCommand("git", "rev-parse", rev).Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Merge attempts to merge the given commits
//
// If there's only one commit, returns it directly.
// Otherwise, tries to merge all commits:
// 1. First attempts a direct merge of all commits
// 2. If direct merge fails with multiple commits, tries two-phase merge:
//   - First merges all commits except the last one
//   - Then tries to merge the last commit
//   - If second phase fails, checks which branches conflict with the last one
func (r *Repo) Merge(message string, base *models.GitRef, commits ...*models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// Try direct merge first
	if ref, fail := r.doMerge(message, base, commits...); ref != nil {
		return ref, nil
	} else if len(commits) <= 1 {
		return nil, fail
	}

	// Try two-phase merge for multiple commits
	previousRef, previousFail := r.doMerge("partial", base, commits[:len(commits)-1]...)
	if previousRef == nil {
		return nil, previousFail
	}

	// Try merging the last commit
	finalRef, finalFail := r.doMerge(message, previousRef, commits[len(commits)-1])
	if finalRef != nil {
		return finalRef, nil
	}

	// Check which branches conflict with the last one
	lastCommit := commits[len(commits)-1]
	for _, commit := range commits[:len(commits)-1] {
		if r.checkConflict(lastCommit, commit) {
			finalFail.ConflictBranches = append(finalFail.ConflictBranches, commit.Name)
		}
	}
	finalFail.ConflictBranches = append(finalFail.ConflictBranches, lastCommit.Name)
	return nil, finalFail
}

// doMerge performs the actual merge operation
func (r *Repo) doMerge(message string, base *models.GitRef, commits ...*models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// Reset to base commit
	if output, err := r.execCommand("git", "checkout", base.Commit).CombinedOutput(); err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git checkout %s", base.Commit),
			Stdout:  string(output),
			Stderr:  err.Error(),
			Status:  "reset failed",
		}
	}
	defer func() {
		// No matter merge success or not, keep the working directory clean
		_ = r.execCommand("git", "reset", "--hard", "HEAD").Run()
	}()
	// Early return for empty commits
	if len(commits) == 0 {
		if output, err := r.execCommand("git", "commit", "--allow-empty", "-m", message).CombinedOutput(); err != nil {
			return nil, &models.GitMergeFailResult{
				Cmdline: fmt.Sprintf("git commit --allow-empty -m %q", message),
				Stdout:  string(output),
				Stderr:  err.Error(),
				Status:  "commit failed",
			}
		}
		hash, err := r.RevParse("HEAD")
		if err != nil {
			return nil, &models.GitMergeFailResult{
				Cmdline: "git rev-parse HEAD",
				Stderr:  err.Error(),
				Status:  "rev-parse failed",
			}
		}
		return &models.GitRef{
			Name:   "HEAD",
			Commit: hash,
		}, nil
	}

	// Prepare merge command
	args := []string{"merge", "--no-ff", "-m", message}
	for _, c := range commits {
		args = append(args, c.Commit)
	}
	cmdline := fmt.Sprintf("git %s", strings.Join(args, " "))

	// Execute merge
	cmd := r.execCommand("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Get conflict details
		conflicts := []models.FileMergeConflict{}
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "CONFLICT ") {
				// Parse conflict details using git diff
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) != 2 {
					continue
				}
				conflictType := strings.Trim(parts[0], "CONFLICT ()")
				path := strings.TrimSpace(strings.Split(parts[1], " in ")[1])

				// Get diff for the conflicted file
				diff, _ := r.execCommand("git", "diff", path).Output()

				conflicts = append(conflicts, models.FileMergeConflict{
					Path:           path,
					ConflictType:   conflictType,
					ConflictDetail: string(diff),
				})
			}
		}

		return nil, &models.GitMergeFailResult{
			Cmdline:     cmdline,
			Stdout:      string(output),
			Stderr:      err.Error(),
			Status:      "merge failed",
			FailedFiles: conflicts,
		}
	}

	// Get resulting commit
	hash, err := r.RevParse("HEAD")
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: cmdline,
			Stderr:  err.Error(),
			Status:  "rev-parse failed",
		}
	}

	return &models.GitRef{
		Name:   "HEAD",
		Commit: hash,
	}, nil
}

// checkConflict checks if two branches have conflicts
func (r *Repo) checkConflict(base, other *models.GitRef) bool {
	// Reset to base commit
	if err := r.execCommand("git", "reset", "--hard", base.Commit).Run(); err != nil {
		return false
	}

	// Try merge without committing
	if err := r.execCommand("git", "merge", "--no-ff", "--no-commit", other.Commit).Run(); err != nil {
		// Clean up
		r.execCommand("git", "merge", "--abort").Run() // Ignore cleanup errors
		return true
	}

	// Clean up
	r.execCommand("git", "merge", "--abort").Run() // Ignore cleanup errors
	return false
}

// GetCommitMessage returns the commit message for the given commit
func (r *Repo) GetCommitMessage(commit string) (string, error) {
	cmd := r.execCommand("git", "log", "-1", "--pretty=format:%B", commit)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit message: %w", err)
	}
	return string(output), nil
}

// EnsureBranch ensures a branch exists and points to the specified commit.
// If the commit is empty, the branch will be deleted.
// If the branch doesn't exist, it will be created.
// If the branch exists but points to a different commit, it will be updated.
func (r *Repo) EnsureBranch(name string, commit string) error {
	var cmd *exec.Cmd
	if commit == "" {
		cmd = r.execCommand("git", "branch", "-D", name)
	} else {
		cmd = r.execCommand("git", "branch", "-f", name, commit)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to ensure branch %s at %s: %s: %w", name, commit, output, err)
	}
	return nil
}

// RefreshRemote fetches the latest changes from the remote repository
func (r *Repo) RefreshRemote() error {
	cmd := r.execCommand("git", "fetch", "--all")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch remote: %s: %w", output, err)
	}
	return nil
}

// PushRemote update a remote branch to a specified commit
func (r *Repo) PushRemote(remote, branch, commit string) error {
	cmd := r.execCommand("git", "push", "-f", remote, fmt.Sprintf("%s:refs/heads/%s", commit, branch))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push to remote %s: %s: %w", remote, output, err)
	}
	return nil
}

// Config set a git config in the repository
func (r *Repo) Config(key, value string) error {
	cmd := r.execCommand("git", "config", "--local", key, value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set git config %s: %s: %w", key, output, err)
	}
	return nil
}

// Path returns the absolute path to the repository
func (r *Repo) Path() string {
	return r.path
}
