interface BlueprintTitleInput {
  repository?: string | null
  product?: string | null
  project?: string | null
  name?: string | null
  executiveSummary?: string | null
}

export function formatBlueprintTitle(input?: string | BlueprintTitleInput | null): string {
  const productName =
    typeof input === 'string'
      ? formatRepositoryName(input)
      : formatRepositoryName(input?.repository) ||
        formatRepositoryName(input?.product) ||
        formatRepositoryName(input?.project) ||
        formatRepositoryName(input?.name) ||
        formatSummarySubject(input?.executiveSummary)

  return productName ? `The ${productName} Blueprint` : 'The Blueprint'
}

function formatRepositoryName(repository?: string | null): string {
  const repoName = repository?.trim().split('/').filter(Boolean).pop()?.trim()
  if (!repoName) return ''

  return repoName
    .replace(/[-_]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .split(' ')
    .map(capitalize)
    .join(' ')
}

function capitalize(word: string): string {
  if (!word) return word
  return word.charAt(0).toUpperCase() + word.slice(1)
}

function formatSummarySubject(summary?: string | null): string {
  const subject = summary?.trim().match(/^(.+?)\s+(?:is|are)\s+(?:a|an|the)\b/i)?.[1]
  return formatRepositoryName(subject)
}
