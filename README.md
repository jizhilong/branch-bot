# branch-bot

branch-bot is a GitLab-based branch management tool to help teams safely merge multiple feature branches.
It operates entirely through GitLab issue comments, providing a lightweight,
conflict-aware approach to managing parallel development workflows.

![](/docs/images/bb-workflow.jpg)

## Key Features

- **GitLab Native Integration**: All operations are performed through GitLab issue comments - no additional UI or CLI tools needed
- **Safe Testing Branch Creation**: Merge multiple feature branches into a single testing branch without affecting the master branch
- **Simple Commands**: Easy-to-use commands for managing branches (`add <branch>`, `remove <branch>`) via issue comments
- **Conflict Detection**: Automatically detect and prevent conflicting changes from entering the testing branch
- **CI/CD Integration**: Built-in support for GitLab CI pipelines and deployment workflows
- **Stateful Management**: Track the state of merged branches and their relationships through GitLab issues

## Why branch-bot?

When multiple teams work on different features in parallel, integrating their changes for testing becomes challenging:

- Manual merging is error-prone and time-consuming
- Conflicts are discovered too late in the development cycle
- Testing environments become unstable due to incompatible changes
- It's difficult to track which features are included in each testing branch

branch-bot solves these problems by providing:

- Automated, conflict-aware branch merging
- Clear visibility into the composition of testing branches
- Easy addition and removal of features from testing branches
- Integration with existing CI/CD pipelines

## Getting Started

### Prerequisites

- GitLab instance (self-hosted or GitLab.com)
- Bot user account with appropriate permissions
- Access to create issues and merge requests

### Basic Usage

1. Create an issue in your GitLab project with the label: **branch-bot**
2. Add branches using the command: `!bb add <branch_name>` or `!bb !<merge_request_iid>`
3. View the current branch-bot status with: `!bb status`

### Available Commands

| Command | Description |
| ------- | ----------- |
| `!bb` | View current branch-bot status |
| `!bb add <branch/!mr-id>` | Add or update a branch/merge request |
| `!bb remove <branch/!mr-id>` | Remove a branch/merge request |
| `!bb reset [--base master]` | Reset branch-bot to specified base branch |
| `!bb fork` | Create new branch-bot issue with current state |

### CI/CD Integration

branch-bot automatically creates branches with the pattern `bb-branches/\d+`. You can configure GitLab CI to run specific jobs for these branches:

```yaml
rules:
  - if: '$CI_COMMIT_BRANCH =~ /bb\-branches\/.*/ && $CI_PIPELINE_SOURCE == "push"'
    when: always
```

## Core Principles

branch-bot is built on several key principles for effective branch management:

1. **Prevent Conflicts, Don't Just Resolve Them**: Focus on code organization and clear ownership to minimize conflicts
2. **Last Update Responsibility**: The last team to update their branch is responsible for resolving conflicts
3. **Non-Destructive Operations**: Failed merges don't corrupt the testing branch
4. **Freedom to Exit**: Any branch can be removed from testing branch at any time
5. **Explicit State**: The final merge commit always shows which branches are included

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

branch-bot was inspired by:

- Ctrip's ["Light Merge Accelerator"](https://cloud.tencent.com/developer/article/1157076) concept
- GitLab's [Merge Trains](https://docs.gitlab.com/ee/ci/pipelines/merge_trains.html) feature, though branch-bot takes a different approach by focusing on testing branch management rather than production merges
