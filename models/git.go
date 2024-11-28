package models

import (
	"fmt"
	"strings"
)

// GitRef represents a git reference (branch or commit)
type GitRef struct {
	Name   string // branch name or commit SHA
	Commit string // commit SHA
}

func (r *GitRef) String() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.Commit)
}

// FileMergeConflict represents a merge conflict in a specific file
type FileMergeConflict struct {
	Path           string // conflicted file path
	ConflictType   string // type of conflict (content, delete, etc)
	ConflictDetail string // detailed conflict information
}

// GitMergeFailResult represents a failed merge operation
type GitMergeFailResult struct {
	Cmdline          string              // the git command that was executed
	Stdout           string              // command stdout
	Stderr           string              // command stderr
	Status           string              // command exit status
	FailedFiles      []FileMergeConflict // files with conflicts
	ConflictBranches []string            // branches that conflict with the new branch
}

// AsMarkdown formats the merge result as markdown
func (r *GitMergeFailResult) AsMarkdown() string {
	messages := []string{}

	// Add merge failure summary
	messages = append(messages, "\n<details><summary>merge failed</summary>\n\n"+
		fmt.Sprintf("**commandline**: \n```\n%s\n```\n\n", r.Cmdline)+
		fmt.Sprintf("**stdout**: \n```\n%s\n```\n\n", r.Stdout)+
		fmt.Sprintf("**stderr**: \n```\n%s\n```\n", r.Stderr)+
		"</details>")

	// Add conflict branches if any
	if len(r.ConflictBranches) > 0 {
		newBranch := r.ConflictBranches[len(r.ConflictBranches)-1]
		conflictBranches := strings.Join(r.ConflictBranches[:len(r.ConflictBranches)-1], ", ")
		messages = append(messages, fmt.Sprintf("\n**and `%s` conflicted branches**: `%s`\n", newBranch, conflictBranches))
	}

	// Add conflict details
	if len(r.FailedFiles) > 0 {
		messages = append(messages, "\n**conflicts**: \n")
		for _, file := range r.FailedFiles {
			if len(file.ConflictDetail) < 20000 {
				messages = append(messages, fmt.Sprintf("\n<details><summary>%s: %s</summary>\n\n```diff\n%s\n```\n</details>",
					file.Path, file.ConflictType, file.ConflictDetail))
			} else {
				messages = append(messages, fmt.Sprintf("\n<details><summary>%s: %s</summary>\n\ndiff too large, not shown\n</details>",
					file.Path, file.ConflictType))
			}
		}
	}

	return strings.Join(messages, "\n")
}
