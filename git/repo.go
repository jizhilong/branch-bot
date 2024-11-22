package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jizhilong/light-merge/models"
)

// Repo represents a Git repository
type Repo struct {
	path string // absolute path to the repository
}

// New creates a new Repo instance
func New(path string) (*Repo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Repo{path: absPath}, nil
}

// RevParse returns the commit hash for the given revision
func (r *Repo) RevParse(rev string) (string, error) {
	cmd := exec.Command("git", "rev-parse", rev)
	cmd.Dir = r.path
	output, err := cmd.Output()
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
func (r *Repo) Merge(base *models.GitRef, commits ...*models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// If there's only one commit, return it directly
	if len(commits) == 0 {
		return base, nil
	}

	// First try direct merge
	ref, fail := r.doMerge(base, commits...)
	if ref != nil {
		return ref, nil
	}

	// If direct merge fails with multiple commits, try two-phase merge
	if len(commits) > 1 {
		// First merge all commits except the last one
		previousRef, previousFail := r.doMerge(base, commits[:len(commits)-1]...)
		if previousRef != nil {
			// Then try to merge the last commit
			finalRef, finalFail := r.doMerge(previousRef, commits[len(commits)-1])
			if finalRef != nil {
				return finalRef, nil
			}
			// If second phase fails, check which branches conflict with the last one
			for _, commit := range commits[:len(commits)-1] {
				if r.checkConflict(commits[len(commits)-1], commit) {
					finalFail.ConflictBranches = append(finalFail.ConflictBranches, commit.Name)
				}
			}
			// Add the new branch as the last conflict branch
			finalFail.ConflictBranches = append(finalFail.ConflictBranches, commits[len(commits)-1].Name)
			return nil, finalFail
		}
		return nil, previousFail
	}

	return nil, fail
}

// doMerge performs the actual merge operation
func (r *Repo) doMerge(base *models.GitRef, commits ...*models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// Reset to base commit
	cmd := exec.Command("git", "reset", "--hard", base.Commit)
	cmd.Dir = r.path
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git reset --hard %s", base.Commit),
			Stdout:  string(output),
			Stderr:  err.Error(),
			Status:  "reset failed",
		}
	}

	// Prepare merge command
	args := []string{"merge", "--no-ff"}
	for _, c := range commits {
		args = append(args, c.Commit)
	}
	cmdline := fmt.Sprintf("git %s", strings.Join(args, " "))

	// Execute merge
	cmd = exec.Command("git", args...)
	cmd.Dir = r.path
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
				diffCmd := exec.Command("git", "diff", path)
				diffCmd.Dir = r.path
				diff, _ := diffCmd.Output()

				conflicts = append(conflicts, models.FileMergeConflict{
					Path:           path,
					ConflictType:   conflictType,
					ConflictDetail: string(diff),
				})
			}
		}

		// Clean up
		cleanCmd := exec.Command("git", "reset", "--hard", base.Commit)
		cleanCmd.Dir = r.path
		cleanCmd.Run() // Ignore cleanup errors

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
	cmd := exec.Command("git", "reset", "--hard", base.Commit)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return false
	}

	// Try merge without committing
	cmd = exec.Command("git", "merge", "--no-ff", "--no-commit", other.Commit)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		// Clean up
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = r.path
		abortCmd.Run() // Ignore cleanup errors
		return true
	}

	// Clean up
	abortCmd := exec.Command("git", "merge", "--abort")
	abortCmd.Dir = r.path
	abortCmd.Run() // Ignore cleanup errors
	return false
}
