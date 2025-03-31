import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock modules BEFORE importing the code-under-test
vi.mock('@actions/core')
vi.mock('@actions/github')
vi.mock('@actions/exec', async (importOriginal) => {
  // Keep original types but mock implementation
  const actual = (await importOriginal()) as typeof import('@actions/exec')
  return {
    ...actual, // Keep actual types/exports
    exec: vi.fn(), // Mock the core exec function
  }
})

// Now import the modules and the code-under-test
import * as core from '@actions/core'
import * as github from '@actions/github'
import * as exec from '@actions/exec'
import { run } from './main' // Assuming run() is exported or the main entry point

// --- Mock Implementations ---

// Mock Octokit client
const mockOctokit = {
  rest: {
    reactions: {
      createForIssueComment: vi.fn(),
    },
    // No issues.createComment mock needed anymore
  },
}

// Type helper for the mocked exec function itself
type MockedExecFn = ReturnType<typeof vi.fn>
const mockedExecFn = exec.exec as MockedExecFn

// --- Test State & Setup ---

// Use an object to store scenario-specific mock behaviors for exec.exec
let execScenarioMocks: Record<
  string,
  (args?: readonly string[], options?: any) => Promise<any>
> = {}

beforeEach(() => {
  // Reset mocks history
  vi.resetAllMocks()
  // Clear scenario mocks for exec
  execScenarioMocks = {}

  // --- Default Mock Behaviors ---

  // Core inputs
  vi.spyOn(core, 'getInput').mockImplementation((name: string) => {
    switch (name) {
      case 'github_token':
        return 'test-token'
      case 'comment_id':
        return '12345'
      case 'pr_branch_ref':
        return 'refs/heads/feature-branch'
      case 'base_branch':
        return 'main'
      case 'api_directory':
        return 'api'
      default:
        return ''
    }
  })

  // GitHub context and Octokit
  vi.spyOn(github, 'getOctokit').mockReturnValue(mockOctokit as any)
  Object.defineProperty(github, 'context', {
    value: {
      repo: { owner: 'test-owner', repo: 'test-repo' },
      issue: { number: 99 },
      actor: 'test-actor',
    },
    writable: true, // Allow reassignment if needed in specific tests
  })

  // Central mock implementation for exec.exec
  mockedExecFn.mockImplementation(async (command, args, options) => {
    const commandString = `${command} ${args?.join(' ') || ''}`.trim()
    core.debug(`Mocked exec call: ${commandString}`)

    // Check if a scenario-specific mock exists for this command
    const scenarioMock = execScenarioMocks[commandString]
    if (scenarioMock) {
      core.debug(`Using scenario mock for: ${commandString}`)
      // Scenario mocks handle return values, stdout simulation, errors
      return await scenarioMock(args, options)
    }

    // Default behavior: success (exit code 0)
    core.debug(`Using default success mock for: ${commandString}`)
    // Simulate stdout if listener is provided (needed for getExecOutput)
    if (options?.listeners?.stdout) {
      options.listeners.stdout(Buffer.from(''))
    }
    return 0
  })
})

afterEach(() => {
  // Verify no unexpected errors
  vi.restoreAllMocks()
})

// --- Test Suite ---
describe('Rebase Bot Action', () => {
  // Helper function to check if exec was called with specific command/args
  const wasExecCalledWith = (command: string, args: string[]): boolean => {
    return mockedExecFn.mock.calls.some(
      (call) =>
        call[0] === command && JSON.stringify(call[1]) === JSON.stringify(args), // Simple array comparison
    )
  }

  it('should succeed and push if rebase has no conflicts', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 0
    }
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async () => 0

    // Act
    await run()

    // Assert
    expect(core.setFailed).not.toHaveBeenCalled()
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'rocket' }))

    // Verify key commands were called / not called
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(true)
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(false)
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(false)
    expect(wasExecCalledWith('git', ['rebase', '--continue'])).toBe(false)
  })

  it('should fail and abort if conflicts exist outside api directory', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('src/other/file.ts\napi/some/generated.file'),
      )
      return 0
    }
    execScenarioMocks['git rebase --abort'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 0
    }

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith(
      expect.stringContaining("Conflicts found outside 'api'"),
    )
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'confused' }))

    // Verify key commands were called / not called
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(true)
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(false)
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(false)
  })

  it('should fail and abort if conflicts include .tsp files within api directory', async () => {
    // Arrange: Simulate failed rebase with .tsp conflict inside api/
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('api/some/generated.file\napi/manual/types.tsp'),
      ) // .tsp conflict
      return 0
    }
    execScenarioMocks['git rebase --abort'] = async (args, options) => {
      return 0
    }

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith(
      expect.stringContaining("Conflicts found in .tsp file(s) within 'api'."),
    )
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'confused' }))

    // Verify key commands were called / not called
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(true)
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(false)
    expect(wasExecCalledWith('git', ['rebase', '--continue'])).toBe(false)
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(false)
  })

  it('should run make gen-api, continue rebase, and push if conflicts are only inside api directory', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('api/some/generated.file\napi/another/spec.yaml'),
      )
      return 0
    }
    execScenarioMocks['git add .'] = async () => 0
    execScenarioMocks['make gen-api'] = async () => 0
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('M  api/some/generated.file\nM  api/another/spec.yaml'),
      )
      return 0
    }
    execScenarioMocks['git rebase --continue'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 0
    }
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async () => 0

    // Act
    await run()

    // Assert
    expect(core.setFailed).not.toHaveBeenCalled()
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'rocket' }))

    // Verify key commands were called / not called
    expect(wasExecCalledWith('git', ['add', '.'])).toBe(true) // Should be called at least once
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--continue'])).toBe(true)
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(false)
  })

  it('should fail and abort if make gen-api does not resolve conflicts', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      options.listeners.stdout(Buffer.from('api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git add .'] = async () => 0
    execScenarioMocks['make gen-api'] = async () => 0
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(Buffer.from('UU api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git rebase --abort'] = async () => 0

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith(
      expect.stringContaining('Conflicts remain after `make gen-api`'),
    )
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'confused' }))

    // Verify key commands were called / not called
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--continue'])).toBe(false)
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(false)
  })

  it('should fail and abort if git rebase --continue fails', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      options.listeners.stdout(Buffer.from('api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git add .'] = async () => 0
    execScenarioMocks['make gen-api'] = async () => 0
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      options.listeners.stdout(Buffer.from('M api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git rebase --continue'] = async (args, options) => {
      expect(options?.ignoreReturnCode).toBe(true)
      return 1
    }
    execScenarioMocks['git rebase --abort'] = async () => 0

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith(
      expect.stringContaining('`git rebase --continue` failed.'),
    )
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'confused' }))

    // Verify key commands were called / not called
    expect(wasExecCalledWith('make', ['gen-api'])).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--continue'])).toBe(true)
    expect(wasExecCalledWith('git', ['rebase', '--abort'])).toBe(true)
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(false)
  })

  it('should fail if push fails', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      return 0
    }
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async () => {
        throw new Error('Simulated push failure')
      }

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith('Simulated push failure')
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledTimes(2)
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: '+1' }))
    expect(
      mockOctokit.rest.reactions.createForIssueComment,
    ).toHaveBeenCalledWith(expect.objectContaining({ content: 'confused' }))

    // Verify key commands were called / not called
    expect(
      wasExecCalledWith('git', [
        'push',
        '--force-with-lease',
        'origin',
        'feature-branch',
      ]),
    ).toBe(true)
  })
})
