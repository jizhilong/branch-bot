package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MergeTrain represents a testing branch and its composition.
//
// Note: Despite the similar name, this is NOT related to GitLab's Merge Train feature.
// This is a tool for managing a testing branch composed of multiple feature branches.
type MergeTrain struct {
	ProjectID  int
	IssueIID   int
	BranchName string
	Members    []MergeTrainItem
}

// MergeTrainItem represents a member branch in merge train
type MergeTrainItem struct {
	ProjectID    int    // GitLab project ID
	Branch       string // branch name
	MergedCommit string // commit that has been merged into bb branch
}

// NewMergeTrain creates a new merge train
func NewMergeTrain(projectID int, issueIID int, branchName string) *MergeTrain {
	return &MergeTrain{
		ProjectID:  projectID,
		IssueIID:   issueIID,
		BranchName: branchName,
		Members:    make([]MergeTrainItem, 0),
	}
}

// AddMember adds a new member to merge train
func (mt *MergeTrain) AddMember(branch, commit string) {
	mt.Members = append(mt.Members, MergeTrainItem{
		ProjectID:    mt.ProjectID,
		Branch:       branch,
		MergedCommit: commit,
	})
}

// RemoveMember removes a member from merge train
func (mt *MergeTrain) RemoveMember(branch string) {
	newMembers := make([]MergeTrainItem, 0, len(mt.Members))
	for _, m := range mt.Members {
		if m.Branch != branch {
			newMembers = append(newMembers, m)
		}
	}
	mt.Members = newMembers
}

// GenerateCommitMessage creates a commit message for the bb branch
func (mt *MergeTrain) GenerateCommitMessage() string {
	data, err := json.MarshalIndent(mt, "", "  ")
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("Light-Merge State\n\n%s", string(data))
}

// GenerateCommitMessageWithNewMemberSet creates a commit message for the bb branch with new members, but don't add them to the merge train
func (mt *MergeTrain) GenerateCommitMessageWithNewMemberSet(newMembers []MergeTrainItem) string {
	originalMembers := mt.Members
	defer func() { mt.Members = originalMembers }()
	mt.Members = newMembers
	return mt.GenerateCommitMessage()
}

// LoadFromCommitMessage parses a commit message to reconstruct a MergeTrain
func LoadFromCommitMessage(message string) (*MergeTrain, error) {
	lines := strings.Split(message, "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "Light-Merge State") {
		return nil, fmt.Errorf("invalid commit message format")
	}

	var mt MergeTrain
	if err := json.Unmarshal([]byte(strings.Join(lines[2:], "\n")), &mt); err != nil {
		return nil, fmt.Errorf("failed to deserialize MergeTrain: %w", err)
	}

	return &mt, nil
}
