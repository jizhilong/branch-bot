package gitlab

import (
	"fmt"
	"github.com/jizhilong/light-merge/models"
	"log/slog"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func (h *Webhook) reply(note *gitlab.IssueCommentEvent, message string) {
	_, _, err := h.gl.Notes.CreateIssueNote(note.ProjectID, note.Issue.IID, &gitlab.CreateIssueNoteOptions{
		Body: &message,
	})
	if err != nil {
		slog.Error("Failed to reply to comment", "error", err)
	}
}

func (h *Webhook) awardEmojiAgainstError(note *gitlab.IssueCommentEvent, err error) {
	if err == nil {
		go h.awardEmoji(note, ":white_check_mark:")
	} else {
		go h.awardEmoji(note, ":x:")
	}
}

func (h *Webhook) awardEmoji(note *gitlab.IssueCommentEvent, emoji string) {
	_, _, err := h.gl.AwardEmoji.CreateIssuesAwardEmojiOnNote(note.ProjectID, note.Issue.IID,
		note.ObjectAttributes.ID,
		&gitlab.CreateAwardEmojiOptions{Name: emoji})
	if err != nil {
		slog.Error("Failed to award emoji", "error", err)
	}
}

type MergeRequestLookupError struct {
	mrId int
	err  string
}

func (e MergeRequestLookupError) Error() string {
	return fmt.Sprintf("failed to get merge request %d: %s", e.mrId, e.err)
}

func (h *Webhook) revParseRemote(projectId int, branchName string) (*models.GitRef, error) {
	if strings.HasPrefix(branchName, "!") {
		mrIdStr := strings.TrimPrefix(branchName, "!")
		mrId, err := strconv.Atoi(mrIdStr)
		if err != nil {
			return nil, MergeRequestLookupError{
				mrId: mrId,
				err:  fmt.Sprintf("invalid merge request ID: %s", err.Error()),
			}
		}
		mr, _, err := h.gl.MergeRequests.GetMergeRequest(projectId, mrId, nil)
		if err != nil {
			return nil, MergeRequestLookupError{
				mrId: mrId,
				err:  fmt.Sprintf("failed to get merge request: %s", err.Error()),
			}
		}
		return &models.GitRef{Name: mr.SourceBranch, Commit: mr.DiffRefs.HeadSha}, nil
	} else {
		branch, _, err := h.gl.Branches.GetBranch(projectId, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to get branch: %w", err)
		}
		return &models.GitRef{Name: branchName, Commit: branch.Commit.ID}, nil
	}
}
