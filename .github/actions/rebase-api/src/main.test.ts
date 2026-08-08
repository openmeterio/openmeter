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
    issues: {
      getComment: vi.fn(),
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

  // Mock core.setFailed - essential for test assertions
  vi.spyOn(core, 'setFailed').mockImplementation(vi.fn())
  vi.spyOn(core, 'info').mockImplementation(vi.fn())
  vi.spyOn(core, 'error').mockImplementation(vi.fn())
  vi.spyOn(core, 'warning').mockImplementation(vi.fn())
  vi.spyOn(core, 'debug').mockImplementation(vi.fn())
  vi.spyOn(core, 'group').mockImplementation(async (name, fn) => await fn())

  // GitHub context and Octokit
  vi.spyOn(github, 'getOctokit').mockReturnValue(mockOctokit as any)
  // Default mock for getComment - successful fetch with trigger phrase
  mockOctokit.rest.issues.getComment.mockResolvedValue({
    data: {
      id: 12345,
      body: 'This comment contains the trigger: /rebase-api',
    },
  })

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
    // Use console.log for immediate visibility during test run, core.debug might be buffered
    console.log(`Mocked exec call: [${commandString}]`) // Log the command string being checked

    const scenarioMock = execScenarioMocks[commandString]
    if (scenarioMock) {
      console.log(`-> Using scenario mock for: [${commandString}]`)
      try {
        // Scenario mock is responsible for return value (exit code), stdout, or throwing
        const result = await scenarioMock(args, options)
        console.log(
          `-> Scenario mock for [${commandString}] returned: ${result}`,
        )
        return result
      } catch (error: any) {
        console.log(
          `-> Scenario mock for [${commandString}] threw error: ${error.message}`,
        )
        throw error // Re-throw error from scenario mock
      }
    }

    // Default behavior IF NO scenario mock exists: success (exit code 0)
    console.log(`-> Using default success mock for: [${commandString}]`)
    // Simulate empty stdout ONLY if a listener is provided (needed for getExecOutput)
    if (options?.listeners?.stdout) {
      console.log(
        ` -> Default mock: Simulating empty stdout for [${commandString}]`,
      )
      options.listeners.stdout(Buffer.from(''))
    }
    return 0 // Default successful exit code
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
    return mockedExecFn.mock.calls.some((call) => {
      // Check if the command matches
      if (call[0] !== command) return false

      // Check if args match - handle case where args might be undefined in the call
      if (!call[1] && !args.length) return true
      if (!call[1]) return false

      // Compare each argument individually to better debug
      if (call[1].length !== args.length) return false

      // Use JSON.stringify for array comparison
      return JSON.stringify(call[1]) === JSON.stringify(args)
    })
  }

  it('should succeed and push if rebase has no conflicts', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      console.log('Scenario: git rebase (success)')
      expect(options?.ignoreReturnCode).toBe(true)
      return 0 // Success exit code
    }
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async (args, options) => {
        console.log('Scenario: git push (success)')
        expect(options?.ignoreReturnCode).toBeFalsy() // pushChanges calls exec directly
        return 0 // Success exit code
      }

    // Act
    await run()

    // Debug output
    console.log('Mock calls:', JSON.stringify(mockedExecFn.mock.calls, null, 2))

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
      console.log('Scenario: git rebase (fail)')
      expect(options?.ignoreReturnCode).toBe(true)
      return 1 // Fail exit code
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      console.log('Scenario: git diff (outputting external conflict)')
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('src/other/file.ts\napi/some/generated.file'),
      )
      return 0 // Command success
    }
    execScenarioMocks['git rebase --abort'] = async (args, options) => {
      console.log('Scenario: git rebase --abort (success)')
      expect(options?.ignoreReturnCode).toBe(true)
      return 0 // Abort success
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
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async (args, options) => {
      console.log('Scenario: git rebase (fail)')
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      console.log('Scenario: git diff (outputting tsp conflict)')
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('api/some/generated.file\napi/manual/types.tsp'),
      ) // .tsp conflict
      return 0
    }
    execScenarioMocks['git rebase --abort'] = async (args, options) => {
      console.log('Scenario: git rebase --abort (success)')
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
      console.log('Scenario: git rebase (fail)')
      return 1
    }
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      console.log('Scenario: git diff (outputting internal conflicts)')
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('api/some/generated.file\napi/another/spec.yaml'),
      )
      return 0
    }
    execScenarioMocks['git add .'] = async () => {
      console.log('Scenario: git add .')
      return 0
    }
    execScenarioMocks['make gen-api'] = async () => {
      console.log('Scenario: make gen-api')
      return 0
    }
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      console.log('Scenario: git status (clean)')
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(
        Buffer.from('M  api/some/generated.file\nM  api/another/spec.yaml'),
      )
      return 0
    }
    execScenarioMocks['git rebase --continue'] = async (args, options) => {
      console.log('Scenario: git rebase --continue (success)')
      expect(options?.ignoreReturnCode).toBe(true)
      return 0
    }
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async () => {
        console.log('Scenario: git push (success)')
        return 0
      }

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
    expect(wasExecCalledWith('git', ['add', '.'])).toBe(true)
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
    execScenarioMocks['git rebase origin/main'] = async () => 1
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      console.log('Scenario: git diff (internal conflict)')
      options.listeners.stdout(Buffer.from('api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git add .'] = async () => {
      console.log('Scenario: git add .')
      return 0
    }
    execScenarioMocks['make gen-api'] = async () => {
      console.log('Scenario: make gen-api')
      return 0
    }
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      console.log('Scenario: git status (UU conflict)')
      expect(options?.listeners?.stdout).toBeDefined()
      options.listeners.stdout(Buffer.from('UU api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git rebase --abort'] = async () => {
      console.log('Scenario: git rebase --abort')
      return 0
    }

    // Act
    await run()

    // Assert
    expect(core.setFailed).toHaveBeenCalledWith(
      expect.stringContaining(
        'Conflicts still present after running make gen-api.',
      ),
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
  })

  it('should fail and abort if git rebase --continue fails', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async () => 1
    execScenarioMocks['git diff --name-only --diff-filter=U'] = async (
      args,
      options,
    ) => {
      console.log('Scenario: git diff (internal conflict)')
      options.listeners.stdout(Buffer.from('api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git add .'] = async () => {
      console.log('Scenario: git add .')
      return 0
    }
    execScenarioMocks['make gen-api'] = async () => {
      console.log('Scenario: make gen-api')
      return 0
    }
    execScenarioMocks['git status --porcelain'] = async (args, options) => {
      console.log('Scenario: git status (clean)')
      options.listeners.stdout(Buffer.from('M api/spec.yaml'))
      return 0
    }
    execScenarioMocks['git rebase --continue'] = async (args, options) => {
      console.log('Scenario: git rebase --continue (fail)')
      expect(options?.ignoreReturnCode).toBe(true)
      return 1
    }
    execScenarioMocks['git rebase --abort'] = async () => {
      console.log('Scenario: git rebase --abort')
      return 0
    }

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
  })

  it('should fail if push fails', async () => {
    // Arrange
    execScenarioMocks['git rebase origin/main'] = async () => 0
    execScenarioMocks['git push --force-with-lease origin feature-branch'] =
      async (args, options) => {
        console.log('Scenario: git push (fail - throw)')
        expect(options?.ignoreReturnCode).toBeFalsy()
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
