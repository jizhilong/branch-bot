package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jizhilong/light-merge/core"
)

const (
	issueIDFile = ".git/LM_ISSUE_IID"
)

func main() {
	// Get current directory as git repo
	repoPath, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get current directory", "error", err)
		os.Exit(1)
	}

	// Parse command line arguments
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: lm-local <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  add <branch>    Add or update a branch in merge train")
		fmt.Println("  remove <branch> Remove a branch from merge train")
		os.Exit(1)
	}

	// Get or create issue ID
	issueID, err := getOrCreateIssueID(repoPath)
	if err != nil {
		slog.Error("Failed to get/create issue ID", "error", err)
		os.Exit(1)
	}

	// Create operator
	operator, err := core.LoadMergeTrainOperator(0, issueID, repoPath)
	if err != nil {
		slog.Error("Failed to load merge train operator", "error", err)
		os.Exit(1)
	}

	// Execute command
	switch args[0] {
	case "add":
		if len(args) != 2 {
			fmt.Println("Usage: lm-local add <branch>")
			os.Exit(1)
		}
		result, fail := operator.Add(args[1])
		if fail != nil {
			fmt.Println("Failed to add branch:", fail.AsMarkdown())
			os.Exit(1)
		}
		fmt.Printf("Successfully added branch %s at commit %s\n", args[1], result.Commit)

	case "remove":
		if len(args) != 2 {
			fmt.Println("Usage: lm-local remove <branch>")
			os.Exit(1)
		}
		result, fail := operator.Remove(args[1])
		if fail != nil {
			fmt.Println("Failed to remove branch:", fail.AsMarkdown())
			os.Exit(1)
		}
		if result != nil {
			fmt.Printf("Successfully removed branch %s, new HEAD is %s\n", args[1], result.Commit)
		} else {
			fmt.Printf("Successfully removed branch %s, merge train is now empty\n", args[1])
		}

	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func getOrCreateIssueID(repoPath string) (int, error) {
	// Try to read existing issue ID
	idPath := filepath.Join(repoPath, issueIDFile)
	data, err := os.ReadFile(idPath)
	if err == nil {
		// File exists, parse ID
		id, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return 0, fmt.Errorf("invalid issue ID in file: %w", err)
		}
		return id, nil
	}

	// File doesn't exist, create new ID
	if !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to read issue ID file: %w", err)
	}

	id := 1
	if err := os.WriteFile(idPath, []byte(strconv.Itoa(id)), 0644); err != nil {
		return 0, fmt.Errorf("failed to write issue ID file: %w", err)
	}

	return id, nil
}
