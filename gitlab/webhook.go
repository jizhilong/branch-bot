package gitlab

import (
	"errors"
	"fmt"
	"github.com/jizhilong/light-merge/core"
	"github.com/jizhilong/light-merge/git"
	"github.com/jizhilong/light-merge/models"
	"github.com/xanzy/go-gitlab"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Command interface {
	CommandName() string
	String() string
}

type AddCommand struct {
	BranchName string
}

func (c *AddCommand) CommandName() string {
	return "add"
}

func (c *AddCommand) String() string {
	return fmt.Sprintf("%s %s", c.CommandName(), c.BranchName)
}

type RemoveCommand struct {
	BranchName string
}

func (c *RemoveCommand) String() string {
	return fmt.Sprintf("%s %s", c.CommandName(), c.BranchName)
}

func (c *RemoveCommand) CommandName() string {
	return "remove"
}

// ParseCommand parses a command from issue comment
func ParseCommand(comment string) (Command, error) {
	// Expected format: !lm <command> [args...]
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, "!lm ") {
		return nil, fmt.Errorf("invalid command format")
	}
	comment = strings.TrimPrefix(comment, "!lm ")
	parts := strings.Split(comment, " ")
	if len(parts) < 1 {
		return nil, fmt.Errorf("missing command")
	}
	commandName := strings.TrimSpace(parts[0])
	switch commandName {
	case "add", "remove":
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid number of arguments, expected 1 branch name")
		}
		if commandName == "add" {
			return &AddCommand{BranchName: parts[1]}, nil
		} else {
			return &RemoveCommand{BranchName: parts[1]}, nil
		}
	default:
		return nil, fmt.Errorf("unknown command")
	}
}

// Webhook handles HTTP requests for light-merge
type Webhook struct {
	// port is the port number to listen on
	port int
	// repoDir is the directory where repositories are stored
	repoDir string
	// glToken is the GitLab access token
	glToken string
	// branchNamePrefix is the prefix for the output branch name
	branchNamePrefix string
	// gl is the GitLab client
	gl *gitlab.Client
}

// NewWebhook creates a new server instance
func NewWebhook(gitlabUrl, gitlabToken, repoDir, branchNamePrefix string, port int) (*Webhook, error) {
	if port <= 0 {
		return nil, errors.New("invalid port number")
	}
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}
	gl, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}
	return &Webhook{
		port:             port,
		repoDir:          repoDir,
		glToken:          gitlabToken,
		branchNamePrefix: branchNamePrefix,
		gl:               gl,
	}, nil
}

// Start starts the HTTP server
func (h *Webhook) Start() error {
	http.HandleFunc("/webhook", h.handleWebhook)
	return http.ListenAndServe(fmt.Sprintf(":%d", h.port), nil)
}

// handleWebhook handles GitLab webhook events
func (h *Webhook) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// always return 200 OK
	defer func() {
		w.WriteHeader(http.StatusOK)
	}()
	if r.Method != http.MethodPost {
		return
	}

	event, err := h.parseGitlabEvent(r)
	if err != nil {
		slog.Error("Failed to parse GitLab event", "error", err)
		return
	}

	switch e := event.(type) {
	case *gitlab.IssueCommentEvent:
		if e.User.Bot {
			return
		}
		cmd, err := ParseCommand(e.ObjectAttributes.Note)
		if err != nil {
			slog.Error("Invalid command", "error", err)
			return
		}
		if cmd != nil {
			h.HandleCommand(cmd, e)
		}
	default:
		slog.Warn("Unknown event type", "type", fmt.Sprintf("%T", e))
		return
	}
}

func (h *Webhook) parseGitlabEvent(r *http.Request) (interface{}, error) {
	defer func() {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			log.Printf("could discard request body: %v", err)
		}
		if err := r.Body.Close(); err != nil {
			log.Printf("could not close request body: %v", err)
		}
	}()

	event := r.Header.Get("X-Gitlab-Event")
	if strings.TrimSpace(event) == "" {
		return nil, errors.New("missing X-Gitlab-Event Header")
	}

	eventType := gitlab.EventType(event)
	if !isEventSubscribed(eventType) {
		return nil, errors.New("event not defined to be parsed")
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		return nil, errors.New("error reading request body")
	}

	return gitlab.ParseWebhook(eventType, payload)
}

func (h *Webhook) HandleCommand(cmd Command, event *gitlab.IssueCommentEvent) {
	logger := slog.With(
		"project_id", event.ProjectID,
		"issue_id", event.Issue.IID,
	)
	logger.Info("Handling command", "command", cmd.String())

	operator, err := h.getOperator(event.ProjectID, event.Issue.IID,
		event.Project.PathWithNamespace, event.Project.GitHTTPURL)
	if err != nil {
		logger.Error("Failed to get operator", "error", err)
		return
	}

	switch c := cmd.(type) {
	case *AddCommand:
		logger = logger.With("branch", c.BranchName)
		ref, err := h.revParseRemote(event.ProjectID, c.BranchName)
		if err != nil {
			logger.Error("Failed to get remote ref", "error", err)
			go h.reply(event, fmt.Sprintf("branch %s not found.", c.BranchName))
			return
		}
		result, fail := operator.AddAndPush(ref)
		if fail != nil {
			logger.Error("Failed to add branch", "error", fail.AsMarkdown())
			go h.reply(event, fail.AsMarkdown())
			return
		}
		logger.Info("Successfully added branch", "result", result)
		go h.awardEmoji(event, ":white_check_mark:")
	case *RemoveCommand:
		logger = logger.With("branch", c.BranchName)
		ref, err := h.revParseRemote(event.ProjectID, c.BranchName)
		if err != nil {
			logger.Error("Failed to get remote ref", "error", err)
			go h.reply(event, fmt.Sprintf("branch %s not found.", c.BranchName))
			return
		}
		result, fail := operator.RemoveAndPush(ref.Name)
		if fail != nil {
			logger.Error("Failed to remove branch", "error", fail.AsMarkdown())
			go h.reply(event, fail.AsMarkdown())
			return
		}
		logger.Info("Successfully removed branch", "result", result)
		go h.awardEmoji(event, ":white_check_mark:")
	default:
		logger.Error("Unknown command")
	}
}

func (h *Webhook) getOperator(projectId, issueIID int, pathWithNameSpace, projectUrl string) (*core.MergeTrainOperator, error) {
	if repo, err := h.syncRepo(pathWithNameSpace, projectUrl); err != nil {
		return nil, fmt.Errorf("failed to sync repository: %w", err)
	} else {
		branchName := fmt.Sprintf("%s%d", h.branchNamePrefix, issueIID)
		return core.LoadMergeTrainOperator(repo, branchName, projectId, issueIID)
	}
}

func (h *Webhook) reply(note *gitlab.IssueCommentEvent, message string) {
	_, _, err := h.gl.Notes.CreateIssueNote(note.ProjectID, note.Issue.IID, &gitlab.CreateIssueNoteOptions{
		Body: &message,
	})
	if err != nil {
		slog.Error("Failed to reply to comment", "error", err)
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

func (h *Webhook) revParseRemote(projectId int, branchName string) (*models.GitRef, error) {
	if strings.HasPrefix(branchName, "!") {
		mrIdStr := strings.TrimPrefix(branchName, "!")
		mrId, err := strconv.Atoi(mrIdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid merge request ID: %w", err)
		}
		mr, _, err := h.gl.MergeRequests.GetMergeRequest(projectId, mrId, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get merge request: %w", err)
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

func (h *Webhook) syncRepo(pathWithNameSpace, projectUrl string) (*git.Repo, error) {
	repoPath := fmt.Sprintf("%s/%s", h.repoDir, pathWithNameSpace)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create repo directory: %w", err)
		}
	}
	gitDirPath := fmt.Sprintf("%s/.git", repoPath)
	var repo *git.Repo
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		// clone from gitlab
		u, err := url.Parse(projectUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse project URL: %w", err)
		}
		cloneUrl := fmt.Sprintf("%s://light-merge:%s@%s%s", u.Scheme, h.glToken, u.Host, u.Path)
		repo, err = git.Clone(cloneUrl, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
		if err = repo.Config("user.name", "light-merge"); err != nil {
			return nil, fmt.Errorf("failed to set user name: %w", err)
		}
		if err = repo.Config("user.email", "operator@light-merge.localhost"); err != nil {
			return nil, fmt.Errorf("failed to set user email: %w", err)
		}
	} else {
		repo, err = git.New(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository: %w", err)
		}
		err = repo.RefreshRemote()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh remote: %w", err)
		}
	}
	return repo, nil
}

func isEventSubscribed(event gitlab.EventType) bool {
	acceptedEvents := []gitlab.EventType{
		gitlab.EventTypePush,
		gitlab.EventTypeNote,
	}
	for _, e := range acceptedEvents {
		if event == e {
			return true
		}
	}
	return false
}
