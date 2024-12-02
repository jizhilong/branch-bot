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

// MergeTrainViewHelper provides helper functions for convert merge train to merge train views
type MergeTrainViewHelper interface {
	// URL generators
	BranchURL(projectID int, branchName string) string
	CommitURL(projectID int, commitSHA string) string

	// Branch information
	GetBranchLatestCommit(projectID int, branchName string) (*models.CommitView, error)
	// MergeRequest information
	GetMergeRequestInfo(projectID int, branchName string) (*models.MergeRequestView, error)

	// Save merge train view to storage
	Save(*models.MergeTrainView) error
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
func (o *MergeTrainOperator) AddAndPush(ref *models.GitRef) (*models.GitRef, error) {
	// Add the branch to the merge train
	mergeResult, fail := o.Add(ref)
	if fail != nil {
		return nil, fail
	}

	// Push the changes
	err := o.repo.PushRemote("origin", o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, err
	}

	return mergeResult, nil
}

// Add adds or updates a branch in the merge train
func (o *MergeTrainOperator) Add(ref *models.GitRef) (*models.GitRef, error) {
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
	message := o.mergeTrain.GenerateCommitMessageWithNewMemberSet(newMembers)

	// Try to merge all branches with the generated message
	mergeResult, mergeErr := o.repo.Merge(message, refs[0], refs[1:]...)
	if mergeErr != nil {
		return nil, mergeErr
	}

	// Only update merge train state if merge was successful
	o.mergeTrain.Members = newMembers

	// Create or update the light-merge branch
	err := o.repo.EnsureBranch(o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, err
	}

	return mergeResult, nil
}

// RemoveAndPush removes a branch from the merge train and pushes the changes
func (o *MergeTrainOperator) RemoveAndPush(branchName string) (*models.GitRef, error) {
	// Remove the branch from the merge train
	mergeResult, fail := o.Remove(branchName)
	if fail != nil {
		return nil, fail
	}

	var pushCommit string
	// If mergeResult is nil, it means the merge train is empty after removal, set pushCommit to empty string will delete the remote result branch
	if mergeResult == nil {
		pushCommit = ""
	} else {
		pushCommit = mergeResult.Commit
	}

	// Push the changes
	err := o.repo.PushRemote("origin", o.mergeTrain.BranchName, pushCommit)
	if err != nil {
		return nil, err
	}

	return mergeResult, nil
}

// Remove removes a branch from the merge train and updates the light-merge branch
func (o *MergeTrainOperator) Remove(branchName string) (*models.GitRef, error) {
	// Check if branch exists in merge train
	var branchIndex = -1
	for i, member := range o.mergeTrain.Members {
		if member.Branch == branchName {
			branchIndex = i
			break
		}
	}
	if branchIndex == -1 {
		return nil, fmt.Errorf("branch %s is not a member of merge train", branchName)
	}

	// Create a copy of current members without the branch to remove
	currentMembers := make([]models.MergeTrainItem, 0, len(o.mergeTrain.Members)-1)
	currentMembers = append(currentMembers, o.mergeTrain.Members[:branchIndex]...)
	currentMembers = append(currentMembers, o.mergeTrain.Members[branchIndex+1:]...)

	// If no members left after removal, remove local result branch and return nil
	if len(currentMembers) == 0 {
		err := o.repo.EnsureBranch(o.mergeTrain.BranchName, "")
		if err != nil {
			return nil, err
		}
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
	message := o.mergeTrain.GenerateCommitMessageWithNewMemberSet(currentMembers)

	// Try to merge remaining branches
	mergeResult, mergeErr := o.repo.Merge(message, refs[0], refs[1:]...)
	if mergeErr != nil {
		return nil, mergeErr
	}

	// Update merge train state
	o.mergeTrain.Members = currentMembers

	// Update the light-merge branch
	err := o.repo.EnsureBranch(o.mergeTrain.BranchName, mergeResult.Commit)
	if err != nil {
		return nil, err
	}

	return mergeResult, nil
}

// SyncMergeTrainView synchronizes the merge train view with the actual state
func (o *MergeTrainOperator) SyncMergeTrainView(helper MergeTrainViewHelper) error {
	view, err := o.getMergeTrainView(helper)
	if err != nil {
		return err
	}

	return helper.Save(view)
}

// getMergeTrainView returns a view of the merge train
func (o *MergeTrainOperator) getMergeTrainView(helper MergeTrainViewHelper) (*models.MergeTrainView, error) {
	mt, repo := o.mergeTrain, o.repo
	view := &models.MergeTrainView{
		Branch:  mt.BranchName,
		URL:     helper.BranchURL(mt.ProjectID, mt.BranchName),
		Members: make([]models.MemberView, 0, len(mt.Members)),
	}
	if len(mt.Members) == 0 {
		return view, nil
	}

	// Get light-merge branch latest commit
	trainCommit, err := repo.RevParse(mt.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get light-merge branch commit: %w", err)
	}
	view.Commit = &models.CommitView{
		SHA: trainCommit,
		URL: helper.CommitURL(mt.ProjectID, trainCommit),
	}

	// Convert members
	for _, member := range mt.Members {
		memberView := models.MemberView{
			Branch:    member.Branch,
			BranchURL: helper.BranchURL(mt.ProjectID, member.Branch),
		}

		// Set merged commit info
		if member.MergedCommit != "" {
			memberView.MergedCommit = &models.CommitView{
				SHA: member.MergedCommit,
				URL: helper.CommitURL(mt.ProjectID, member.MergedCommit),
			}
		}

		// Get latest commit
		latestCommit, err := helper.GetBranchLatestCommit(mt.ProjectID, member.Branch)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest commit for branch %s: %w", member.Branch, err)
		}
		memberView.LatestCommit = latestCommit

		// Get merge request info if exists
		if mr, err := helper.GetMergeRequestInfo(mt.ProjectID, member.Branch); err == nil {
			memberView.MergeRequest = mr
		}

		view.Members = append(view.Members, memberView)
	}

	return view, nil
}
