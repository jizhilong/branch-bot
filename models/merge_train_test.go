package models

import (
	"testing"
)

func TestGenerateAndLoadCommitMessage(t *testing.T) {
	// Create a new MergeTrain and add members
	mtOriginal := NewMergeTrain(123, 456, "bb-branches/1")
	mtOriginal.AddMember("feature-1", "abc123")
	mtOriginal.AddMember("feature-2", "def456")

	// Generate commit message
	message := mtOriginal.GenerateCommitMessage()

	// Load from commit message
	mtLoaded, err := LoadFromCommitMessage(message)
	if err != nil {
		t.Fatalf("LoadFromCommitMessage() error = %v", err)
	}

	// Compare original and loaded MergeTrain
	if mtOriginal.ProjectID != mtLoaded.ProjectID || mtOriginal.IssueIID != mtLoaded.IssueIID || mtOriginal.BranchName != mtLoaded.BranchName {
		t.Errorf("Loaded MergeTrain does not match original: got %v, want %v", mtLoaded, mtOriginal)
	}

	if len(mtOriginal.Members) != len(mtLoaded.Members) {
		t.Fatalf("Loaded MergeTrain has different number of members: got %d, want %d", len(mtLoaded.Members), len(mtOriginal.Members))
	}

	for i, member := range mtOriginal.Members {
		if member != mtLoaded.Members[i] {
			t.Errorf("Loaded member does not match original: got %v, want %v", mtLoaded.Members[i], member)
		}
	}

	// Test after removing a member
	mtOriginal.RemoveMember("feature-1")
	message = mtOriginal.GenerateCommitMessage()

	mtLoaded, err = LoadFromCommitMessage(message)
	if err != nil {
		t.Fatalf("LoadFromCommitMessage() after RemoveMember error = %v", err)
	}

	// Compare original and loaded MergeTrain after removal
	if len(mtOriginal.Members) != len(mtLoaded.Members) {
		t.Fatalf("Loaded MergeTrain after removal has different number of members: got %d, want %d", len(mtLoaded.Members), len(mtOriginal.Members))
	}

	for i, member := range mtOriginal.Members {
		if member != mtLoaded.Members[i] {
			t.Errorf("Loaded member after removal does not match original: got %v, want %v", mtLoaded.Members[i], member)
		}
	}
}
