import * as core from '@actions/core'
import * as github from '@actions/github'
import * as exec from '@actions/exec'
import { GitHub } from '@actions/github/lib/utils' // Helper for Octokit typing
import path from 'path' // Needed for path manipulation

// Define expected input types for clarity
interface ActionInputs {
  githubToken: string
  commentId: number
  prBranchRef: string
  baseBranch: string
  apiDirectory: string
}

/**
 * Helper function to get typed inputs
 */
function getInputs(): ActionInputs {
  const githubToken = core.getInput('github_token', { required: true })
  // comment_id comes from the event payload, but let's get it via input for consistency
  const commentId = parseInt(
    core.getInput('comment_id', { required: true }),
    10,
  )
  const prBranchRef = core.getInput('pr_branch_ref', { required: true })
  const baseBranch = core.getInput('base_branch', { required: true })
  const apiDirectory = core.getInput('api_directory', { required: true })

  if (isNaN(commentId)) {
    throw new Error('Input "comment_id" is not a valid integer.')
  }

  return { githubToken, commentId, prBranchRef, baseBranch, apiDirectory }
}

/**
 * Adds a reaction to the triggering comment.
 */
async function addReaction(
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
    // Continue execution even if reaction fails
  }
}

/**
 * Executes a shell command and returns the trimmed stdout.
 * Throws error if command fails.
 */
async function getExecOutput(
  command: string,
  args?: string[],
): Promise<string> {
  let output = ''
  const options: exec.ExecOptions = {
    listeners: {
      stdout: (data: Buffer) => {
        output += data.toString()
      },
    },
    ignoreReturnCode: false, // Throw on non-zero exit code
  }
  await exec.exec(command, args, options)
  return output.trim()
}

/**
 * Executes a shell command and returns the exit code.
 * Does not throw on non-zero exit code.
 */
async function tryExec(
  command: string,
  args?: string[],
  options?: exec.ExecOptions,
): Promise<number> {
  return await exec.exec(command, args, { ...options, ignoreReturnCode: true })
}

/**
 * The main function for the action.
 */
export async function run(): Promise<void> {
  let finalReaction: 'rocket' | 'confused' = 'confused'
  let failureCode: string | null = null // Specific failure code (e.g., 'CONFLICT_OUTSIDE_API')
  let octokit: InstanceType<typeof GitHub> | null = null
  let inputs: ActionInputs | null = null

  try {
    core.info('Starting Rebase API Action...')
    inputs = getInputs()
    octokit = github.getOctokit(inputs.githubToken)
    const prBranchName = inputs.prBranchRef.replace('refs/heads/', '')

    // 1. Add initial reaction
    await addReaction(octokit, inputs.commentId, '+1')

    core.info('Configuring Git user...')
    await exec.exec('git', ['config', 'user.name', 'github-actions[bot]'])
    await exec.exec('git', [
      'config',
      'user.email',
      'github-actions[bot]@users.noreply.github.com',
    ])

    core.info(`Fetching latest changes for ${inputs.baseBranch}...`)
    await exec.exec('git', ['fetch', 'origin', inputs.baseBranch])

    core.info(
      `Attempting to rebase ${prBranchName} onto origin/${inputs.baseBranch}...`,
    )
    const rebaseExitCode = await tryExec('git', [
      'rebase',
      `origin/${inputs.baseBranch}`,
    ])

    if (rebaseExitCode !== 0) {
      core.warning('Rebase failed. Checking for conflicts...')
      const conflictOutput = await getExecOutput('git', [
        'diff',
        '--name-only',
        '--diff-filter=U',
      ])
      const conflictingFiles = conflictOutput
        .split('\n')
        .filter((f) => f.length > 0)

      if (conflictingFiles.length === 0) {
        // Should not happen if rebaseExitCode != 0, but safety check
        throw new Error('Rebase failed but no conflicting files found.')
      }

      core.info(`Conflicting files: ${conflictingFiles.join(', ')}`)

      // Check if any conflict is OUTSIDE the apiDirectory
      const conflictOutsideApi = conflictingFiles.some(
        (file) =>
          !path
            .normalize(file)
            .startsWith(path.normalize(inputs!.apiDirectory + '/')),
      )

      // Check if any conflicting file IS a .tsp file
      const conflictInTspFile = conflictingFiles.some((file) =>
        file.endsWith('.tsp'),
      )

      if (conflictOutsideApi) {
        core.error(
          `Conflicts detected outside the specified API directory ('${inputs.apiDirectory}'). Aborting.`,
        )
        failureCode = 'CONFLICT_OUTSIDE_API'
        await tryExec('git', ['rebase', '--abort']) // Abort the rebase
        throw new Error(
          `Rebase failed: Conflicts found outside '${inputs.apiDirectory}'.`,
        )
      } else if (conflictInTspFile) {
        core.error(
          `Conflicts detected in non-generated .tsp file(s) within '${inputs.apiDirectory}'. Aborting.`,
        )
        failureCode = 'CONFLICT_IN_TSP'
        await tryExec('git', ['rebase', '--abort']) // Abort the rebase
        throw new Error(
          `Rebase failed: Conflicts found in .tsp file(s) within '${inputs.apiDirectory}'.`,
        )
      } else {
        core.info(
          'Conflicts are only within the API directory and not in .tsp files. Proceeding with auto-resolution...',
        )
        // Stage conflicts as-is
        await exec.exec('git', ['add', '.'])

        core.info('Running make gen-api to resolve conflicts...')
        await exec.exec('make', ['gen-api']) // Assuming 'make gen-api' exists and works

        core.info('Staging potentially updated API files...')
        await exec.exec('git', ['add', '.'])

        // Check if 'make gen-api' actually resolved the conflicts
        const statusOutput = await getExecOutput('git', [
          'status',
          '--porcelain',
        ])
        if (statusOutput.includes('UU ')) {
          core.error('Conflicts still present after running make gen-api.')
          failureCode = 'API_RESOLUTION_FAILED'
          await tryExec('git', ['rebase', '--abort'])
          throw new Error('Conflicts remain after `make gen-api`.')
        }

        core.info('Committing resolved API changes...')
        // Use --no-edit to avoid opening an editor if the rebase was partially completed
        // Use commit --amend if the conflict happened on the very first commit of the rebase sequence?
        // Let's use `git rebase --continue` which handles the commit automatically.
        const continueExitCode = await tryExec('git', ['rebase', '--continue'])
        if (continueExitCode !== 0) {
          core.error(
            '`git rebase --continue` failed after resolving conflicts.',
          )
          failureCode = 'REBASE_CONTINUE_FAILED'
          // Attempt abort again, although state might be complex
          await tryExec('git', ['rebase', '--abort'])
          throw new Error('`git rebase --continue` failed.')
        }
        core.info(
          'Rebase continued successfully after API conflict resolution.',
        )
      }
    }

    core.info('Rebase successful or conflicts resolved.')
    core.info(`Force-pushing updated branch ${prBranchName}...`)
    // Use --force-with-lease for safety
    await exec.exec('git', [
      'push',
      '--force-with-lease',
      'origin',
      prBranchName,
    ])

    core.info('Branch pushed successfully.')
    finalReaction = 'rocket' // Mark as success
  } catch (error: any) {
    core.error(`Action failed: ${error.message}`)
    if (error.stack) {
      core.debug(error.stack)
    }
    // failureCode might have been set already for specific cases
    finalReaction = 'confused'
    core.setFailed(error.message)
  } finally {
    if (octokit && inputs) {
      // 4. Add final reaction
      await addReaction(octokit, inputs.commentId, finalReaction)
    }
    core.info('Rebase API Action finished.')
  }
}

// Run the action
run()
