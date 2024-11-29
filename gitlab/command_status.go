package gitlab

import (
	"github.com/jizhilong/light-merge/core"
	"github.com/xanzy/go-gitlab"
	"log/slog"
)

type StatusCommand string

func (c StatusCommand) CommandName() string {
	return "status"
}

func (c StatusCommand) String() string {
	return "status"
}

func (c StatusCommand) Process(h *Webhook, event *gitlab.IssueCommentEvent, logger *slog.Logger, operator *core.MergeTrainOperator) {
	err := operator.SyncMergeTrainView(&MergeTrainViewGlHelper{gl: h.gl, event: event})
	if err != nil {
		logger.Error("Failed to sync merge train view", "error", err)
		go h.reply(event, "failed to sync merge train view")
		return
	}
}
