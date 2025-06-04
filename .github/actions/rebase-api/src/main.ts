import * as core from '@actions/core'
import * as github from '@actions/github'
import { GitHub } from '@actions/github/lib/utils'
import { ActionInputs } from './types'
import { getInputs } from './utils'
import { addReaction, getCommentBody } from './github'
import {
  configureGitUser,
  fetchBranch,
  attemptRebase,
  handleRebaseConflicts,
  pushChanges,
} from './git'

// --- Main Exported Function ---

/**
 * Main function that orchestrates the rebase action.
 */
export async function run(): Promise<void> {
  let finalReaction: 'rocket' | 'confused' = 'confused' // Start assuming failure
  let octokit: InstanceType<typeof GitHub> | null = null
  let inputs: ActionInputs | null = null

  try {
    core.info('Starting Rebase API Action...')
    inputs = getInputs()
    octokit = github.getOctokit(inputs.githubToken)

    // --- Check comment body ---
    const commentBody = await getCommentBody(octokit, inputs.commentId)
    if (!commentBody || !commentBody.startsWith(inputs.triggerPhrase)) {
      core.info(
        `Comment does not contain trigger phrase "${inputs.triggerPhrase}". Skipping rebase.`,
      )
      // No reaction needed here, just exit gracefully.
      // We could set finalReaction to something else if required, but skipping seems fine.
      return
    }
    core.info('Trigger phrase found in comment.')
    // --- End Check ---

    const prBranchName = inputs.prBranchRef.replace('refs/heads/', '')

    await addReaction(octokit, inputs.commentId, '+1')

    await configureGitUser()
    await fetchBranch(inputs.baseBranch)

    const rebaseExitCode = await attemptRebase(inputs.baseBranch)

    if (rebaseExitCode !== 0) {
      await handleRebaseConflicts(inputs)
      // If handleRebaseConflicts succeeds, we log it.
      // If it throws, execution jumps to the catch block.
      core.info('Conflicts handled successfully.')
    }

    // If we reach here, rebase was successful or conflicts were resolved.
    core.info('Rebase successful or conflicts resolved.')
    await pushChanges(prBranchName)

    finalReaction = 'rocket' // Mark success only if push succeeds
    core.info('Action completed successfully.')
  } catch (error: any) {
    core.error(`Action failed: ${error.message}`) // Log the specific error that was caught
    if (error.stack) {
      core.debug(error.stack)
    }
    core.setFailed(error.message) // Mark the action as failed using the caught error message
    // finalReaction remains 'confused'
  } finally {
    // Add final reaction regardless of success/failure
    if (octokit && inputs) {
      await addReaction(octokit, inputs.commentId, finalReaction)
    }
    core.info('Rebase API Action finished.')
  }
}

// Execute the function when this file is run directly (not when imported)
if (require.main === module) {
  run().catch((error) => {
    console.error('Unhandled error in action:', error)
    process.exit(1)
  })
}
