/**
 * Input parameters for the action
 */
export interface ActionInputs {
  githubToken: string
  commentId: number
  prBranchRef: string
  baseBranch: string
  apiDirectory: string
  triggerPhrase: string
}
