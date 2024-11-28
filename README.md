# Light-Merge

Light-Merge is a GitLab-based branch management tool that helps teams safely merge multiple feature branches into a testing branch. It operates entirely through GitLab issue comments, providing a lightweight, conflict-aware approach to managing parallel development workflows.

![Light Merge Workflow](/docs/images/light-merge-workflow.jpg)

## Key Features

- **GitLab Native Integration**: All operations are performed through GitLab issue comments - no additional UI or CLI tools needed
- **Safe Testing Branch Creation**: Merge multiple feature branches into a single testing branch without affecting the master branch
- **Simple Commands**: Easy-to-use commands for managing branches (`add <branch>`, `remove <branch>`) via issue comments
- **Conflict Detection**: Automatically detect and prevent conflicting changes from entering the testing branch
- **CI/CD Integration**: Built-in support for GitLab CI pipelines and deployment workflows
- **Stateful Management**: Track the state of merged branches and their relationships through GitLab issues

## Why Light-Merge?

When multiple teams work on different features in parallel, integrating their changes for testing becomes challenging:

- Manual merging is error-prone and time-consuming
- Conflicts are discovered too late in the development cycle
- Testing environments become unstable due to incompatible changes
- It's difficult to track which features are included in each testing branch

Light-Merge solves these problems by providing:

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

1. Create an issue in your GitLab project with the label: **light-merge**
2. Add branches using the command: `!lm add <branch_name>` or `!lm !<merge_request_iid>`
3. View the current light-merge status with: `!lm status`

### Available Commands

| Command | Description |
| ------- | ----------- |
| `!lm` | View current light-merge status |
| `!lm add <branch/!mr-id>` | Add or update a branch/merge request |
| `!lm remove <branch/!mr-id>` | Remove a branch/merge request |
| `!lm reset [--base master]` | Reset light-merge to specified base branch |
| `!lm fork` | Create new light-merge issue with current state |

### CI/CD Integration

Light-Merge automatically creates branches with the pattern `light-merges/\d+`. You can configure GitLab CI to run specific jobs for these branches:

```yaml
rules:
  - if: '$CI_COMMIT_BRANCH =~ /light\-merges\/.*/ && $CI_PIPELINE_SOURCE == "push"'
    when: always
```

## Core Principles

Light-Merge is built on several key principles for effective branch management:

1. **Prevent Conflicts, Don't Just Resolve Them**: Focus on code organization and clear ownership to minimize conflicts
2. **Last Update Responsibility**: The last team to update their branch is responsible for resolving conflicts
3. **Non-Destructive Operations**: Failed merges don't corrupt the testing branch
4. **Freedom to Exit**: Any branch can be removed from testing branch at any time
5. **Explicit State**: The final merge commit always shows which branches are included

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Light-Merge was inspired by:

- Ctrip's ["Light Merge Accelerator"](https://cloud.tencent.com/developer/article/1157076) concept
- GitLab's [Merge Trains](https://docs.gitlab.com/ee/ci/pipelines/merge_trains.html) feature, though Light-Merge takes a different approach by focusing on testing branch management rather than production merges
