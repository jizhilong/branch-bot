package core

import (
	"fmt"

	"github.com/jizhilong/light-merge/git"
	"github.com/jizhilong/light-merge/models"
)

// MergeTrainOperator handles operations on a merge train
type MergeTrainOperator struct {
	repo       *git.Repo
	mergeTrain *models.MergeTrain
}

// LoadMergeTrainOperator loads or creates a merge train operator
func LoadMergeTrainOperator(projectID int64, issueIID int, repoPath string) (*MergeTrainOperator, error) {
	// Initialize git repo
	repo, err := git.New(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Try to load merge train from branch
	branchName := fmt.Sprintf("auto/light-merge-%d", issueIID)
	commit, err := repo.RevParse(branchName)
	if err != nil {
		// If branch doesn't exist, create a new merge train
		return &MergeTrainOperator{
			repo: repo,
			mergeTrain: &models.MergeTrain{
				ProjectID:  projectID,
				IssueIID:   issueIID,
				BranchName: branchName,
				Members:    make([]models.MergeTrainItem, 0),
			},
		}, nil
	}

	// Get commit message
	message, err := repo.GetCommitMessage(commit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit message: %w", err)
	}

	// Load merge train from commit message
	mergeTrain, err := models.LoadFromCommitMessage(projectID, issueIID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to load merge train from commit message: %w", err)
	}

	return &MergeTrainOperator{
		repo:       repo,
		mergeTrain: mergeTrain,
	}, nil
}

// Add adds or updates a branch in the merge train
func (o *MergeTrainOperator) Add(branchName string) (*models.GitRef, *models.GitMergeFailResult) {
	// Get the commit hash for the branch
	commit, err := o.repo.RevParse(branchName)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git rev-parse %s", branchName),
			Stderr:  err.Error(),
			Status:  "branch not found",
		}
	}

	// Create GitRef for the branch
	ref := &models.GitRef{
		Name:   branchName,
		Commit: commit,
	}

	// Create a copy of current members
	currentMembers := make([]models.MergeTrainItem, len(o.mergeTrain.Members))
	copy(currentMembers, o.mergeTrain.Members)

	// Remove the branch if it's already in the merge train
	for i, member := range currentMembers {
		if member.Branch == branchName {
			// Remove this member
			currentMembers = append(currentMembers[:i], currentMembers[i+1:]...)
			break
		}
	}
	newMembers := append(currentMembers, models.MergeTrainItem{
		ProjectID:    o.mergeTrain.ProjectID,
		Branch:       branchName,
		MergedCommit: commit,
	})

	// Prepare refs for merge
	refs := make([]*models.GitRef, 0, len(currentMembers)+1)
	// Add existing members
	for _, member := range currentMembers {
		refs = append(refs, &models.GitRef{
			Name:   member.Branch,
			Commit: member.MergedCommit,
		})
	}

	// Add the new branch
	refs = append(refs, ref)

	// Generate commit message before merge
	message, err := o.mergeTrain.GenerateCommitMessageWithNewMemberSet(newMembers)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: "generate commit message",
			Stderr:  err.Error(),
			Status:  "internal error",
		}
	}

	// Try to merge all branches with the generated message
	mergeResult, mergeErr := o.repo.Merge(message, refs[0], refs[1:]...)
	if mergeErr != nil {
		return nil, mergeErr
	}

	// Only update merge train state if merge was successful
	o.mergeTrain.Members = newMembers

	// Create or update the light-merge branch
	err = o.repo.EnsureBranch(o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git branch -f %s %s", o.mergeTrain.BranchName, mergeResult.Commit),
			Stderr:  err.Error(),
			Status:  "failed to update branch",
		}
	}

	return mergeResult, nil
}
