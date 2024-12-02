package gitlab

import (
	"fmt"
	"github.com/jizhilong/light-merge/core"
	"github.com/xanzy/go-gitlab"
	"log/slog"
)

type RemoveCommand struct {
	BranchName string
}

func (c *RemoveCommand) String() string {
	return fmt.Sprintf("%s %s", c.CommandName(), c.BranchName)
}

func (c *RemoveCommand) CommandName() string {
	return "remove"
}

func (c *RemoveCommand) Process(h *Webhook, event *gitlab.IssueCommentEvent, logger *slog.Logger, operator *core.MergeTrainOperator) {
	logger = logger.With("branch", c.BranchName)
	ref, err := h.revParseRemote(event.ProjectID, c.BranchName)
	if err != nil {
		logger.Error("Failed to get remote ref", "error", err)
		go h.reply(event, fmt.Sprintf("branch %s not found.", c.BranchName))
		return
	}
	result, fail := operator.RemoveAndPush(ref.Name)
	if fail != nil {
		logger.Error("Failed to remove branch", "error", fail)
	} else {
		logger.Info("Successfully removed branch", "result", result)
	}
	h.awardEmojiAgainstError(event, fail)
	err = operator.SyncMergeTrainView(&MergeTrainViewGlHelper{gl: h.gl, event: event, err: fail})
	if err != nil {
		logger.Error("Failed to sync merge train view", "error", err)
		go h.reply(event, "failed to sync merge train view")
		return
	}
}
