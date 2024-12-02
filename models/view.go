package models

import (
	"fmt"
	"strings"
)

// view.go contains types for rendering merge train status.
// These view types are used to generate human-readable outputs like
// mermaid graphs and markdown tables in GitLab issue comments.

// MergeTrainView represents a merge train with display information
type MergeTrainView struct {
	Branch  string
	URL     string
	Commit  *CommitView
	Members []MemberView
}

// MemberView represents a member branch with display information
type MemberView struct {
	Branch       string
	BranchURL    string
	MergeRequest *MergeRequestView // optional, only if branch is from MR
	MergedCommit *CommitView       // commit that has been merged
	LatestCommit *CommitView       // latest commit on branch
}

// MergeRequestView contains merge request display information
type MergeRequestView struct {
	IID    int
	Title  string
	URL    string
	Author string
}

// CommitView contains commit display information
type CommitView struct {
	SHA string // full SHA
	URL string
}

// MarkdownAble is an interface for types that can be rendered as markdown
type MarkdownAble interface {
	AsMarkdown() string
}

// RenderMermaid generates a mermaid graph representation
func (v *MergeTrainView) RenderMermaid() string {
	if len(v.Members) == 0 {
		return "this light merge train is empty."
	}

	// Start mermaid graph
	graph := []string{
		"graph LR",
	}

	// Add nodes and edges
	for idx, m := range v.Members {
		// Format branch name and commit
		name := m.Branch
		if m.MergeRequest != nil {
			name = fmt.Sprintf("!%d - %s", m.MergeRequest.IID, m.MergeRequest.Title)
			// Escape quotes in title to prevent mermaid syntax errors
			name = strings.ReplaceAll(name, "\"", "'")
		}

		// Format commit hash
		commit := "null"
		if m.MergedCommit != nil {
			commit = m.MergedCommit.SHA[:8]
		}

		// For first node, add light-merge node definition
		if idx == 0 {
			lmNode := fmt.Sprintf("LM[(\"%s(%s)\")]", v.Branch, v.Commit.SHA[:8])
			graph = append(graph, fmt.Sprintf("m%d(\"%s\") -- %s --> %s;", idx, name, commit, lmNode))
		} else {
			graph = append(graph, fmt.Sprintf("m%d(\"%s\") -- %s --> LM;", idx, name, commit))
		}
	}

	// Add click events for links
	graph = append(graph, fmt.Sprintf("click LM \"%s\" _blank", v.URL))
	for idx, m := range v.Members {
		url := m.BranchURL
		if m.MergeRequest != nil {
			url = m.MergeRequest.URL
		}
		graph = append(graph, fmt.Sprintf("click m%d \"%s\" _blank", idx, url))
	}

	return fmt.Sprintf("```mermaid\n%s\n```", strings.Join(graph, "\n"))
}

// RenderTable generates a markdown table representation
func (v *MergeTrainView) RenderTable() string {
	if len(v.Members) == 0 {
		return ""
	}

	// Table header
	table := []string{
		"| Branch | Merge Request | Merged Commit | Latest Commit | Note |",
		"| ------ | ------------ | ------------- | ------------- | ---- |",
	}

	// Add light-merge branch status
	trainCommit := "null"
	if v.Commit != nil {
		trainCommit = fmt.Sprintf("[%s](%s)", v.Commit.SHA[:8], v.Commit.URL)
	}
	table = append(table, fmt.Sprintf("| [%s](%s) | null | null | %s |  |", v.Branch, v.URL, trainCommit))

	// Add member branches
	for _, m := range v.Members {
		branch := fmt.Sprintf("[%s](%s)", m.Branch, m.BranchURL)

		mr := "null"
		if m.MergeRequest != nil {
			mr = fmt.Sprintf("[!%d(%s): %s](%s)",
				m.MergeRequest.IID,
				m.MergeRequest.Author,
				m.MergeRequest.Title,
				m.MergeRequest.URL)
		}

		merged := "null"
		if m.MergedCommit != nil {
			merged = fmt.Sprintf("[%s](%s)", m.MergedCommit.SHA[:8], m.MergedCommit.URL)
		}

		latest := "null"
		if m.LatestCommit != nil {
			latest = fmt.Sprintf("[%s](%s)", m.LatestCommit.SHA[:8], m.LatestCommit.URL)
		}

		// Add update hint if needed
		hint := ""
		if m.LatestCommit != nil && (m.MergedCommit == nil || m.LatestCommit.SHA != m.MergedCommit.SHA) {
			hint = fmt.Sprintf("Update to latest: `!lm add %s`", m.Branch)
		}

		table = append(table, fmt.Sprintf("| %s | %s | %s | %s | %s |", branch, mr, merged, latest, hint))
	}

	return strings.Join(table, "\n")
}
