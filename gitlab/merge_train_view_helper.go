package gitlab

import (
	"fmt"
	"github.com/jizhilong/light-merge/models"
	"github.com/xanzy/go-gitlab"
	"log/slog"
)

type MergeTrainViewGlHelper struct {
	gl    *gitlab.Client
	event *gitlab.IssueCommentEvent
}

func (m MergeTrainViewGlHelper) BranchURL(projectID int, branchName string) string {
	if projectID == m.event.ProjectID {
		return fmt.Sprintf("%s/-/tree/%s", m.event.Project.WebURL, branchName)
	}
	project, _, err := m.gl.Projects.GetProject(projectID, nil)
	if err != nil {
		slog.Error("failed to get project", "projectID", projectID, "error", err)
		return ""
	}
	return fmt.Sprintf("%s/-/tree/%s", project.WebURL, branchName)
}

func (m MergeTrainViewGlHelper) CommitURL(projectID int, commitSHA string) string {
	if projectID == m.event.ProjectID {
		return fmt.Sprintf("%s/-/commit/%s", m.event.Project.WebURL, commitSHA)
	}
	project, _, err := m.gl.Projects.GetProject(projectID, nil)
	if err != nil {
		slog.Error("failed to get project", "projectID", projectID, "error", err)
		return ""
	}
	return fmt.Sprintf("%s/-/commit/%s", project.WebURL, commitSHA)
}

func (m MergeTrainViewGlHelper) GetBranchLatestCommit(projectID int, branchName string) (*models.CommitView, error) {
	branch, _, err := m.gl.Branches.GetBranch(projectID, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch %s: %w", branchName, err)
	}
	return &models.CommitView{
		SHA: branch.Commit.ID,
		URL: branch.Commit.WebURL,
	}, nil
}

func (m MergeTrainViewGlHelper) GetMergeRequestInfo(projectID int, branchName string) (*models.MergeRequestView, error) {
	mrList, _, err := m.gl.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &branchName,
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}
	if len(mrList) == 0 {
		return nil, nil
	} else {
		mr := mrList[0]
		return &models.MergeRequestView{
			IID:    mr.IID,
			URL:    mr.WebURL,
			Author: mr.Author.Username,
			Title:  mr.Title,
		}, nil
	}
}

func (m MergeTrainViewGlHelper) Save(view *models.MergeTrainView) error {
	description := fmt.Sprintf("%s\n%s", view.RenderMermaid(), view.RenderTable())
	_, _, err := m.gl.Issues.UpdateIssue(m.event.ProjectID, m.event.Issue.IID, &gitlab.UpdateIssueOptions{
		Description: &description,
	})
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	} else {
		return nil
	}
}
