package models

// GitRef represents a git reference (branch or commit)
type GitRef struct {
	Name   string // branch name or commit SHA
	Commit string // commit SHA
}

// MergeFailure represents a merge conflict
type MergeFailure struct {
	Path         string // conflicted file path
	ConflictType string // type of conflict (content, delete, etc)
	Detail       string // detailed conflict information
}

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success      bool           // whether merge succeeded
	Ref          *GitRef        // if success, the resulting ref
	Failures     []MergeFailure // if failed, the conflicts
	FailedBranch string         // if failed, which branch caused the failure
}
