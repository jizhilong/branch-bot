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
func LoadMergeTrainOperator(repo *git.Repo, branchName string, projectID, issueIID int) (*MergeTrainOperator, error) {
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
	mergeTrain, err := models.LoadFromCommitMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to load merge train from commit message: %w", err)
	}

	return &MergeTrainOperator{
		repo:       repo,
		mergeTrain: mergeTrain,
	}, nil
}

// AddAndPush adds a branch to the merge train and pushes the changes
func (o *MergeTrainOperator) AddAndPush(ref *models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// Add the branch to the merge train
	mergeResult, fail := o.Add(ref)
	if fail != nil {
		return nil, fail
	}

	// Push the changes
	err := o.repo.PushRemote("origin", o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git push origin %s", o.mergeTrain.BranchName),
			Stderr:  err.Error(),
			Status:  "failed to push",
		}
	}

	return mergeResult, nil
}

// Add adds or updates a branch in the merge train
func (o *MergeTrainOperator) Add(ref *models.GitRef) (*models.GitRef, *models.GitMergeFailResult) {
	// Create a copy of current members
	currentMembers := make([]models.MergeTrainItem, len(o.mergeTrain.Members))
	copy(currentMembers, o.mergeTrain.Members)

	// Remove the branch if it's already in the merge train
	for i, member := range currentMembers {
		if member.Branch == ref.Name {
			// Remove this member
			currentMembers = append(currentMembers[:i], currentMembers[i+1:]...)
			break
		}
	}
	newMembers := append(currentMembers, models.MergeTrainItem{
		ProjectID:    o.mergeTrain.ProjectID,
		Branch:       ref.Name,
		MergedCommit: ref.Commit,
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

// RemoveAndPush removes a branch from the merge train and pushes the changes
func (o *MergeTrainOperator) RemoveAndPush(branchName string) (*models.GitRef, *models.GitMergeFailResult) {
	// Remove the branch from the merge train
	mergeResult, fail := o.Remove(branchName)
	if fail != nil {
		return nil, fail
	}

	// Push the changes
	err := o.repo.PushRemote("origin", o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("git push origin %s", o.mergeTrain.BranchName),
			Stderr:  err.Error(),
			Status:  "failed to push",
		}
	}

	return mergeResult, nil
}

// Remove removes a branch from the merge train and updates the light-merge branch
func (o *MergeTrainOperator) Remove(branchName string) (*models.GitRef, *models.GitMergeFailResult) {
	// Check if branch exists in merge train
	var branchIndex = -1
	for i, member := range o.mergeTrain.Members {
		if member.Branch == branchName {
			branchIndex = i
			break
		}
	}
	if branchIndex == -1 {
		return nil, &models.GitMergeFailResult{
			Cmdline: fmt.Sprintf("check branch %s", branchName),
			Stderr:  "branch not found in merge train",
			Status:  "not found",
		}
	}

	// Create a copy of current members without the branch to remove
	currentMembers := make([]models.MergeTrainItem, 0, len(o.mergeTrain.Members)-1)
	currentMembers = append(currentMembers, o.mergeTrain.Members[:branchIndex]...)
	currentMembers = append(currentMembers, o.mergeTrain.Members[branchIndex+1:]...)

	// If no members left after removal, return nil
	if len(currentMembers) == 0 {
		o.mergeTrain.Members = currentMembers
		return nil, nil
	}

	// Prepare refs for merge
	refs := make([]*models.GitRef, 0, len(currentMembers))
	for _, member := range currentMembers {
		refs = append(refs, &models.GitRef{
			Name:   member.Branch,
			Commit: member.MergedCommit,
		})
	}

	// Generate commit message
	message, err := o.mergeTrain.GenerateCommitMessageWithNewMemberSet(currentMembers)
	if err != nil {
		return nil, &models.GitMergeFailResult{
			Cmdline: "generate commit message",
			Stderr:  err.Error(),
			Status:  "internal error",
		}
	}

	// Try to merge remaining branches
	mergeResult, mergeErr := o.repo.Merge(message, refs[0], refs[1:]...)
	if mergeErr != nil {
		return nil, mergeErr
	}

	// Update merge train state
	o.mergeTrain.Members = currentMembers

	// Update the light-merge branch
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
