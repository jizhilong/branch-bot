package core

import (
	"testing"

	"github.com/jizhilong/light-merge/git"
	"github.com/jizhilong/light-merge/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeTrainOperator_Add(t *testing.T) {
	testRepo := git.NewTestRepo(t)

	operator := &MergeTrainOperator{
		repo: &testRepo.Repo,
		mergeTrain: &models.MergeTrain{
			ProjectID:  123,
			IssueIID:   456,
			BranchName: "auto/light-merge-456",
			Members:    make([]models.MergeTrainItem, 0),
		},
	}

	// Get base commit
	baseHash, err := testRepo.RevParse("HEAD")
	require.NoError(t, err)
	base := &models.GitRef{Name: "main", Commit: baseHash}

	t.Run("add first branch", func(t *testing.T) {
		// Create and add first branch
		_ = testRepo.CreateBranch(base, "feature1", "file1.txt", "feature1 content")
		result, fail := operator.Add("feature1")
		assert.NotNil(t, result)
		assert.Nil(t, fail)
		assert.Len(t, operator.mergeTrain.Members, 1)
		assert.Equal(t, "feature1", operator.mergeTrain.Members[0].Branch)
	})

	t.Run("add second branch without conflict", func(t *testing.T) {
		// Create and add second branch
		_ = testRepo.CreateBranch(base, "feature2", "file2.txt", "feature2 content")
		result, fail := operator.Add("feature2")
		assert.NotNil(t, result)
		assert.Nil(t, fail)
		assert.Len(t, operator.mergeTrain.Members, 2)
		assert.Equal(t, "feature2", operator.mergeTrain.Members[1].Branch)
	})

	t.Run("update existing branch", func(t *testing.T) {
		// Update feature1 with new content
		testRepo.UpdateBranch("feature1", "file1.txt", "updated content")
		result, fail := operator.Add("feature1")
		assert.NotNil(t, result)
		assert.Nil(t, fail)
		assert.Len(t, operator.mergeTrain.Members, 2)
		// feature1 should be moved to the end
		assert.Equal(t, "feature1", operator.mergeTrain.Members[1].Branch)
	})

	t.Run("add branch with conflict", func(t *testing.T) {
		// Create a branch that conflicts with feature1
		_ = testRepo.CreateBranch(base, "conflict", "file1.txt", "conflicting content")
		result, fail := operator.Add("conflict")
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		// MergeTrain should remain unchanged
		assert.Len(t, operator.mergeTrain.Members, 2)
		assert.NotEmpty(t, fail.FailedFiles)
		assert.Equal(t, "file1.txt", fail.FailedFiles[0].Path)
	})

	t.Run("add branch with non-existent base", func(t *testing.T) {
		result, fail := operator.Add("non-existent-branch")
		assert.Nil(t, result)
		assert.NotNil(t, fail)
		assert.Equal(t, "branch not found", fail.Status)
		// MergeTrain should remain unchanged
		assert.Len(t, operator.mergeTrain.Members, 2)
	})
}
