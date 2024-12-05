package models

import (
	"strings"
	"testing"
)

func TestMergeTrainView_RenderMermaid(t *testing.T) {
	tests := []struct {
		name string
		view MergeTrainView
		want string
	}{
		{
			name: "empty train",
			view: MergeTrainView{},
			want: "this light merge train is empty.",
		},
		{
			name: "single branch",
			view: MergeTrainView{
				Branch: "bb-branches/42",
				URL:    "https://gitlab.com/demo/project/-/tree/bb-branches/42",
				Commit: &CommitView{
					SHA: "f9e8d7c6b5a4321",
					URL: "https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321",
				},
				Members: []MemberView{
					{
						Branch:    "main",
						BranchURL: "https://gitlab.com/demo/project/-/tree/main",
						MergedCommit: &CommitView{
							SHA: "a1b2c3d4e5f6789",
							URL: "https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789",
						},
					},
				},
			},
			want: strings.Join([]string{
				"```mermaid",
				"graph LR",
				`m0("main") -- a1b2c3d4 --> BB[("bb-branches/42(f9e8d7c6)")];`,
				`click BB "https://gitlab.com/demo/project/-/tree/bb-branches/42" _blank`,
				`click m0 "https://gitlab.com/demo/project/-/tree/main" _blank`,
				"```",
			}, "\n"),
		},
		{
			name: "multiple branches with MR",
			view: MergeTrainView{
				Branch: "bb-branches/42",
				URL:    "https://gitlab.com/demo/project/-/tree/bb-branches/42",
				Commit: &CommitView{
					SHA: "f9e8d7c6b5a4321",
					URL: "https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321",
				},
				Members: []MemberView{
					{
						Branch:    "main",
						BranchURL: "https://gitlab.com/demo/project/-/tree/main",
						MergedCommit: &CommitView{
							SHA: "a1b2c3d4e5f6789",
							URL: "https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789",
						},
					},
					{
						Branch:    "feature/auth",
						BranchURL: "https://gitlab.com/demo/project/-/tree/feature/auth",
						MergeRequest: &MergeRequestView{
							IID:    123,
							Title:  "Add user authentication API",
							URL:    "https://gitlab.com/demo/project/-/merge_requests/123",
							Author: "john",
						},
						MergedCommit: &CommitView{
							SHA: "b2c3d4e5f6789a",
							URL: "https://gitlab.com/demo/project/-/commit/b2c3d4e5f6789a",
						},
					},
				},
			},
			want: strings.Join([]string{
				"```mermaid",
				"graph LR",
				`m0("main") -- a1b2c3d4 --> BB[("bb-branches/42(f9e8d7c6)")];`,
				`m1("!123 - Add user authentication API") -- b2c3d4e5 --> BB;`,
				`click BB "https://gitlab.com/demo/project/-/tree/bb-branches/42" _blank`,
				`click m0 "https://gitlab.com/demo/project/-/tree/main" _blank`,
				`click m1 "https://gitlab.com/demo/project/-/merge_requests/123" _blank`,
				"```",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.view.RenderMermaid(); got != tt.want {
				t.Errorf("MergeTrainView.RenderMermaid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeTrainView_RenderTable(t *testing.T) {
	tests := []struct {
		name string
		view MergeTrainView
		want string
	}{
		{
			name: "empty train",
			view: MergeTrainView{},
			want: "",
		},
		{
			name: "single branch",
			view: MergeTrainView{
				Branch: "bb-branches/42",
				URL:    "https://gitlab.com/demo/project/-/tree/bb-branches/42",
				Commit: &CommitView{
					SHA: "f9e8d7c6b5a4321",
					URL: "https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321",
				},
				Members: []MemberView{
					{
						Branch:    "main",
						BranchURL: "https://gitlab.com/demo/project/-/tree/main",
						MergedCommit: &CommitView{
							SHA: "a1b2c3d4e5f6789",
							URL: "https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789",
						},
					},
				},
			},
			want: strings.Join([]string{
				"| Branch | Merge Request | Merged Commit | Latest Commit | Note |",
				"| ------ | ------------ | ------------- | ------------- | ---- |",
				"| [bb-branches/42](https://gitlab.com/demo/project/-/tree/bb-branches/42) | null | null | [f9e8d7c6](https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321) |  |",
				"| [main](https://gitlab.com/demo/project/-/tree/main) | null | [a1b2c3d4](https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789) | null |  |",
			}, "\n"),
		},
		{
			name: "branch needs update",
			view: MergeTrainView{
				Branch: "bb-branches/42",
				URL:    "https://gitlab.com/demo/project/-/tree/bb-branches/42",
				Commit: &CommitView{
					SHA: "f9e8d7c6b5a4321",
					URL: "https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321",
				},
				Members: []MemberView{
					{
						Branch:    "feature/auth",
						BranchURL: "https://gitlab.com/demo/project/-/tree/feature/auth",
						MergedCommit: &CommitView{
							SHA: "a1b2c3d4e5f6789",
							URL: "https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789",
						},
						LatestCommit: &CommitView{
							SHA: "b2c3d4e5f6789a",
							URL: "https://gitlab.com/demo/project/-/commit/b2c3d4e5f6789a",
						},
					},
				},
			},
			want: strings.Join([]string{
				"| Branch | Merge Request | Merged Commit | Latest Commit | Note |",
				"| ------ | ------------ | ------------- | ------------- | ---- |",
				"| [bb-branches/42](https://gitlab.com/demo/project/-/tree/bb-branches/42) | null | null | [f9e8d7c6](https://gitlab.com/demo/project/-/commit/f9e8d7c6b5a4321) |  |",
				"| [feature/auth](https://gitlab.com/demo/project/-/tree/feature/auth) | null | [a1b2c3d4](https://gitlab.com/demo/project/-/commit/a1b2c3d4e5f6789) | [b2c3d4e5](https://gitlab.com/demo/project/-/commit/b2c3d4e5f6789a) | Update to latest: `!bb add feature/auth` |",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.view.RenderTable(); got != tt.want {
				t.Errorf("MergeTrainView.RenderTable() = %v, want %v", got, tt.want)
			}
		})
	}
}
