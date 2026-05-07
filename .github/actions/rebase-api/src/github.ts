import * as core from '@actions/core'
import * as github from '@actions/github'
import { GitHub } from '@actions/github/lib/utils'

/**
 * Adds a reaction to a GitHub comment
 */
export async function addReaction(
  octokit: InstanceType<typeof GitHub>,
  commentId: number,
  reaction: '+1' | 'rocket' | 'confused',
): Promise<void> {
  const { owner, repo } = github.context.repo
  core.info(`Adding reaction '${reaction}' to comment ID ${commentId}`)
  try {
    await octokit.rest.reactions.createForIssueComment({
      owner,
      repo,
      comment_id: commentId,
      content: reaction,
    })
  } catch (error: any) {
    core.warning(`Failed to add reaction '${reaction}': ${error.message}`)
  }
}

/**
 * Gets the body of a specific issue comment.
 */
export async function getCommentBody(
  octokit: InstanceType<typeof GitHub>,
  commentId: number,
): Promise<string | null> {
  const { owner, repo } = github.context.repo
  core.info(`Fetching body for comment ID ${commentId}`)
  try {
    const { data: comment } = await octokit.rest.issues.getComment({
      owner,
      repo,
      comment_id: commentId,
    })
    return comment.body || null // Return comment body or null if empty
  } catch (error: any) {
    core.error(`Failed to get comment body: ${error.message}`)
    // Return null or re-throw, depending on desired handling
    // For this use case, failing to get the body should probably stop the action
    throw new Error(`Could not retrieve comment body for ID ${commentId}.`)
  }
}
