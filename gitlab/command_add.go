package gitlab

import (
	"errors"
	"fmt"
	"github.com/jizhilong/branch-bot/core"
	"github.com/xanzy/go-gitlab"
	"log/slog"
)

type AddCommand struct {
	BranchName string
}

func (c *AddCommand) CommandName() string {
	return "add"
}

func (c *AddCommand) String() string {
	return fmt.Sprintf("%s %s", c.CommandName(), c.BranchName)
}

func (c *AddCommand) Process(h *Webhook, event *gitlab.IssueCommentEvent, logger *slog.Logger, operator *core.MergeTrainOperator) {
	logger = logger.With("branch", c.BranchName)
	ref, err := h.revParseRemote(event.ProjectID, c.BranchName)
	var mrLookupErr MergeRequestLookupError
	if err != nil && errors.As(err, &mrLookupErr) {
		logger.Error("Failed to get remote ref", "error", err)
		go h.reply(event, fmt.Sprintf("merge request %s lookup failed ", c.BranchName))
		return
	}
	result, fail := operator.AddAndPush(ref)
	if fail == nil {
		logger.Info("Successfully added branch", "result", result)
	} else {
		logger.Error("Failed to add branch", "error", fail)
	}
	h.awardEmojiAgainstError(event, fail)
	err = operator.SyncMergeTrainView(&MergeTrainViewGlHelper{gl: h.gl, event: event, err: fail})
	if err != nil {
		logger.Error("Failed to sync merge train view", "error", err)
		return
	}
}
