package gitlab

import (
	"errors"
	"fmt"
	"github.com/jizhilong/light-merge/core"
	"github.com/jizhilong/light-merge/git"
	"github.com/xanzy/go-gitlab"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

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

type Command interface {
	CommandName() string
	String() string
	Process(h *Webhook, event *gitlab.IssueCommentEvent, logger *slog.Logger, operator *core.MergeTrainOperator)
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
	case "status":
		return StatusCommand("status"), nil
	default:
		return nil, fmt.Errorf("unknown command")
	}
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
			return
		}
		logger := slog.With(
			"gitlab", h.gl.BaseURL().String(),
			"project_id", e.ProjectID,
			"issue_id", e.Issue.IID,
		)
		operator, err := h.getOperator(e.ProjectID, e.Issue.IID, e.Project.PathWithNamespace, e.Project.GitHTTPURL)
		if err != nil {
			logger.Error("Failed to get operator", "error", err)
			return
		}
		logger.Info("Handling command", "command", cmd.String())
		cmd.Process(h, e, logger, operator)
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
	if eventType != gitlab.EventTypeNote {
		return nil, errors.New("event not defined to be parsed")
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		return nil, errors.New("error reading request body")
	}

	return gitlab.ParseWebhook(eventType, payload)
}

func (h *Webhook) getOperator(projectId, issueIID int, pathWithNameSpace, projectUrl string) (*core.MergeTrainOperator, error) {
	u, err := url.Parse(projectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project URL: %w", err)
	}
	remoteUrl := fmt.Sprintf("%s://light-merge:%s@%s%s", u.Scheme, h.glToken, u.Host, u.Path)
	repoPath := fmt.Sprintf("%s/%s", h.repoDir, pathWithNameSpace)
	if repo, err := git.SyncRepo(repoPath, remoteUrl); err != nil {
		return nil, fmt.Errorf("failed to sync repository: %w", err)
	} else {
		branchName := fmt.Sprintf("%s%d", h.branchNamePrefix, issueIID)
		return core.LoadMergeTrainOperator(repo, branchName, projectId, issueIID)
	}
}
