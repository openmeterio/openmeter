import * as core from '@actions/core'
import * as exec from '@actions/exec'
import { ActionInputs } from './types'

/**
 * Gets action inputs from the environment
 */
export function getInputs(): ActionInputs {
  const githubToken = core.getInput('github_token', { required: true })
  const commentId = parseInt(
    core.getInput('comment_id', { required: true }),
    10,
  )
  const prBranchRef = core.getInput('pr_branch_ref', { required: true })
  const baseBranch = core.getInput('base_branch', { required: true })
  const apiDirectory = core.getInput('api_directory', { required: true })
  const triggerPhrase = core.getInput('trigger_phrase', { required: false })

  if (isNaN(commentId)) {
    throw new Error('Input "comment_id" is not a valid integer.')
  }
  return {
    githubToken,
    commentId,
    prBranchRef,
    baseBranch,
    apiDirectory,
    triggerPhrase,
  }
}

/**
 * Executes a command and returns its output as a string
 */
export async function getExecOutput(
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
 * Executes a command and returns its exit code
 * Does not throw on non-zero exit
 */
export async function tryExec(
  command: string,
  args?: string[],
  options?: exec.ExecOptions,
): Promise<number> {
  return await exec.exec(command, args, { ...options, ignoreReturnCode: true })
}
