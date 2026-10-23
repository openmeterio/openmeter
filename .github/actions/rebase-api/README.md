# Rebase API Action

A GitHub Action that automatically rebases a PR branch against its base branch and handles API-related conflicts intelligently.

## Purpose

This action is designed to simplify the workflow for maintaining PRs that modify generated API code. When triggered (usually by a comment on a PR), it:

1. Rebases the PR branch against the base branch
2. If conflicts occur:
   - Aborts if conflicts are outside the API directory
   - Aborts if conflicts involve `.tsp` files (TypeSpec source files)
   - For other API-related conflicts, uses `make gen-api` to regenerate API code and resolve conflicts
3. Force-pushes the rebased branch back to the repository

## How It Works

The action follows this workflow:

1. Configures the Git user as a GitHub bot
2. Fetches the base branch
3. Attempts to rebase the PR branch onto the base branch
4. If the rebase fails due to conflicts:
   - Analyzes the conflicting files
   - Aborts if conflicts are outside the designated API directory or in `.tsp` files
   - For acceptable conflicts, runs `make gen-api` to regenerate API code
   - Stages the regenerated files and continues the rebase
5. Force-pushes the rebased branch back to the repository
6. Adds a reaction to the triggering comment to indicate success/failure

## Usage

Check example of workflow file in [rebase-api.yml](../workflows/rebase-api.yml).

Note: The workflow requires permissions to write content (for pushing), and to issues/pull-requests (for adding reactions).

## Triggering the Action

Once the workflow is set up, any user with write access to the repository can trigger the action by commenting on a pull request:

```
/rebase-api
```

The action will be triggered if the comment starts with `/rebase-api`.

## Inputs

| Name | Description | Required | Default |
|------|-------------|----------|---------|
| `github_token` | GitHub token for API access | Yes | N/A |
| `comment_id` | ID of the comment that triggered the action | Yes | N/A |
| `pr_branch_ref` | Reference to the PR branch | Yes | N/A |
| `base_branch` | Reference to the base branch | Yes | N/A |
| `api_directory` | Directory containing API code | Yes | N/A |
| `trigger_phrase` | Phrase to trigger the action | No | `/rebase-api` |

## Reactions

The action uses comment reactions to indicate status:
- 👍 (`:+1:`) - Action received and started
- 🚀 (`:rocket:`) - Rebase completed successfully
- 😕 (`:confused:`) - Rebase failed

## Development

To make changes to this action:

1. Edit the TypeScript files in the `src` directory
2. Run tests with `npm test`
3. Run prettier with `npm run format`
