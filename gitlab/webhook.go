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

type Command interface {
	CommandName() string
}

type AddCommand struct {
	BranchName string
}

func (c *AddCommand) CommandName() string {
	return "add"
}

type RemoveCommand struct {
	BranchName string
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
	// gl is the GitLab client
	gl *gitlab.Client
}

// NewServer creates a new server instance
func NewServer(gitlabUrl, gitlabToken, repoDir string, port int) (*Webhook, error) {
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
		port:    port,
		repoDir: repoDir,
		glToken: gitlabToken,
		gl:      gl,
	}, nil
}

// Start starts the HTTP server
func (h *Webhook) Start() error {
	http.HandleFunc("/webhook", h.handleWebhook)
	return http.ListenAndServe(fmt.Sprintf(":%d", h.port), nil)
}

// handleWebhook handles GitLab webhook events
func (h *Webhook) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event, err := h.parseGitlabEvent(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *gitlab.IssueCommentEvent:
		cmd, err := ParseCommand(e.ObjectAttributes.Note)
		if err != nil {
			http.Error(w, "Invalid command", http.StatusBadRequest)
			return
		}
		if cmd != nil {
			h.HandleCommand(cmd, e)
		}
	default:
		http.Error(w, "Event not supported", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
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

	if r.Method != http.MethodPost {
		return nil, errors.New("invalid HTTP Method")
	}

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

	operator, err := h.getOperator(event.ProjectID, event.Issue.IID,
		event.Project.PathWithNamespace, event.Project.GitHTTPURL)
	if err != nil {
		logger.Error("Failed to get operator", "error", err)
		return
	}

	switch c := cmd.(type) {
	case *AddCommand:
		logger = logger.With("branch", c.BranchName)
		result, fail := operator.Add(c.BranchName)
		if fail != nil {
			logger.Error("Failed to add branch", "error", fail.AsMarkdown())
			return
		}
		logger.Info("Successfully added branch", "commit", result.Commit)
	case *RemoveCommand:
		logger = logger.With("branch", c.BranchName)
		result, fail := operator.Remove(c.BranchName)
		if fail != nil {
			logger.Error("Failed to remove branch", "error", fail.AsMarkdown())
			return
		}
		if result != nil {
			logger.Info("Successfully removed branch", "commit", result.Commit)
		} else {
			logger.Info("Successfully removed branch")
		}
	default:
		logger.Error("Unknown command")
	}
}

func (h *Webhook) getOperator(projectId, issueIID int, pathWithNameSpace, projectUrl string) (*core.MergeTrainOperator, error) {
	if repo, err := h.syncRepo(pathWithNameSpace, projectUrl); err != nil {
		return nil, fmt.Errorf("failed to sync repository: %w", err)
	} else {
		return core.LoadMergeTrainOperator(projectId, issueIID, repo.Path())
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
		cloneUrl := fmt.Sprintf("%s://light-merge:%s@%s/%s", u.Scheme, h.glToken, u.Host, u.Path)
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
