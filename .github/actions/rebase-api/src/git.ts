import * as core from '@actions/core'
import * as exec from '@actions/exec'
import path from 'path'
import { ActionInputs } from './types'
import { getExecOutput, tryExec } from './utils'

/**
 * Configures Git user details for GitHub Actions bot.
 */
export async function configureGitUser(): Promise<void> {
  await core.group('Configuring Git user', async () => {
    await exec.exec('git', ['config', 'user.name', 'github-actions[bot]'])
    await exec.exec('git', [
      'config',
      'user.email',
      'github-actions[bot]@users.noreply.github.com',
    ])
  })
}

/**
 * Fetches the specified branch from origin.
 */
export async function fetchBranch(branch: string): Promise<void> {
  await core.group(`Fetching branch '${branch}'`, async () => {
    await exec.exec('git', ['fetch', 'origin', branch])
  })
}

/**
 * Attempts the initial rebase and returns the exit code.
 */
export async function attemptRebase(baseBranch: string): Promise<number> {
  return await core.group('Attempting initial rebase', async () => {
    core.info(`Attempting rebase onto origin/${baseBranch}...`)
    const exitCode = await tryExec('git', ['rebase', `origin/${baseBranch}`])
    if (exitCode !== 0) {
      core.warning('Initial rebase failed, conflicts likely occurred.')
    }
    return exitCode
  })
}

/**
 * Aborts the current rebase operation.
 */
export async function abortRebase(reason: string): Promise<void> {
  core.error(`Aborting rebase: ${reason}`)
  await tryExec('git', ['rebase', '--abort'])
}

/**
 * Gets the list of conflicting files during a rebase.
 */
export async function getConflictingFiles(): Promise<string[]> {
  const conflictOutput = await getExecOutput('git', [
    'diff',
    '--name-only',
    '--diff-filter=U',
  ])
  const files = conflictOutput.split('\n').filter((f) => f.length > 0)
  if (files.length === 0) {
    // This case should ideally not be reached if rebase failed, but handle defensively.
    const reason = 'Rebase command failed but no conflicting files found.'
    core.error(reason) // Log error before potential abort
    await tryExec('git', ['rebase', '--abort']) // Attempt abort directly
    throw new Error(reason)
  }
  core.info(`Conflicting files: ${files.join(', ')}`)
  return files
}

/**
 * Attempts to resolve API conflicts using make gen-api and continue rebase.
 */
export async function resolveApiConflicts(): Promise<void> {
  await core.group('Attempting automatic API conflict resolution', async () => {
    core.info('Staging conflicting files...')
    // Stage *everything* including the unresolved conflicts initially
    await exec.exec('git', ['add', '.'])

    core.info('Running make gen-api to resolve conflicts...')
    try {
      await exec.exec('make', ['gen-api'])
    } catch (error: any) {
      const reason = `'make gen-api' command failed: ${error.message}`
      await abortRebase(reason) // Abort first
      throw new Error(reason) // Then throw
    }

    core.info('Staging potentially updated API files...')
    await exec.exec('git', ['add', '.'])

    // Check if conflicts were ACTUALLY resolved by make gen-api
    const statusOutput = await getExecOutput('git', ['status', '--porcelain'])
    if (statusOutput.includes('UU ')) {
      const reason = 'Conflicts still present after running make gen-api.'
      await abortRebase(reason)
      throw new Error(reason)
    }
    core.info('Conflicts appear resolved by make gen-api.')

    // Continue the rebase
    core.info('Continuing rebase...')
    const continueExitCode = await tryExec('git', ['rebase', '--continue'])
    if (continueExitCode !== 0) {
      const reason = '`git rebase --continue` failed.'
      await abortRebase(reason)
      throw new Error(reason)
    }
    core.info('Rebase continued successfully.')
  })
}

/**
 * Analyzes conflicts and decides whether to abort or attempt resolution.
 */
export async function handleRebaseConflicts(
  inputs: ActionInputs,
): Promise<void> {
  await core.group('Handling rebase conflicts', async () => {
    const conflictingFiles = await getConflictingFiles()

    const conflictOutsideApi = conflictingFiles.some(
      (file) =>
        !path
          .normalize(file)
          .startsWith(path.normalize(inputs.apiDirectory + '/')),
    )
    const conflictInTspFile = conflictingFiles.some((file) =>
      file.endsWith('.tsp'),
    )

    if (conflictOutsideApi) {
      const reason = `Conflicts found outside '${inputs.apiDirectory}'.`
      await abortRebase(reason) // Abort
      throw new Error(reason) // Then throw
    }

    if (conflictInTspFile) {
      const reason = `Conflicts found in .tsp file(s) within '${inputs.apiDirectory}'.`
      await abortRebase(reason)
      throw new Error(reason)
    }

    // If we reach here, conflicts are acceptable for auto-resolution
    core.info(
      'Conflicts are only within the API directory and not in .tsp files.',
    )
    await resolveApiConflicts()
  })
}

/**
 * Force-pushes the current branch to origin.
 */
export async function pushChanges(branchName: string): Promise<void> {
  await core.group('Pushing changes', async () => {
    core.info(`Force-pushing updated branch ${branchName}...`)
    await exec.exec('git', ['push', '--force-with-lease', 'origin', branchName])
    core.info('Branch pushed successfully.')
  })
}
