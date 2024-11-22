package models

import "fmt"

// MergeTrain represents a testing branch and its composition.
//
// Note: Despite the similar name, this is NOT related to GitLab's Merge Train feature.
// This is a tool for managing a testing branch composed of multiple feature branches.
type MergeTrain struct {
	ProjectID  int64
	IssueIID   int
	BranchName string
	Members    []MergeTrainItem
}

// MergeTrainItem represents a member branch in merge train
type MergeTrainItem struct {
	ProjectID    int64  // GitLab project ID
	Branch       string // branch name
	MergedCommit string // commit that has been merged into light-merge branch
}

// NewMergeTrain creates a new merge train
func NewMergeTrain(projectID int64, issueIID int) *MergeTrain {
	return &MergeTrain{
		ProjectID:  projectID,
		IssueIID:   issueIID,
		BranchName: fmt.Sprintf("auto/light-merge-%d", issueIID),
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
