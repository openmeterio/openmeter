import type { SdkOperation } from './sdk-operations.js'
import { namespaceNames } from './sdk-operations.js'

export interface ReadmeResource {
  resource: string
  ops: SdkOperation[]
}

/** GitHub's heading-to-anchor slug: lowercase, strip punctuation, spaces to
 * hyphens. The table of contents is built with this same function so its links
 * always resolve to the headings it points at. */
function slug(heading: string): string {
  return heading
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
}

function tocEntry(heading: string, depth: number): string {
  const indent = '  '.repeat(depth)
  return `${indent}- [${heading}](#${slug(heading)})`
}

/** The fully-qualified call path for an operation, e.g.
 * `customers.credits.grants.create` from the namespace getter, the nested
 * sub-client segments, and the method name. */
function callPath(getter: string, op: SdkOperation): string {
  return [getter, ...op.nestPath, op.methodName].join('.')
}

function summaryCell(op: SdkOperation): string {
  if (!op.summary) {
    return ''
  }
  return op.summary
    .trim()
    .replace(/\s+/g, ' ')
    .replace(/\\/g, '\\\\')
    .replace(/\|/g, '\\|')
}

function operationsTable(getter: string, ops: SdkOperation[]): string {
  const header = ['| Method | HTTP | Description |', '| --- | --- | --- |']
  const rows = ops.map((op) => {
    const call = `\`client.${callPath(getter, op)}\``
    const http = `\`${op.verb.toUpperCase()} ${op.path}\``
    return `| ${call} | ${http} | ${summaryCell(op)} |`
  })
  return [...header, ...rows].join('\n')
}

const HEADINGS = {
  toc: 'Table of Contents',
  install: 'Installation',
  init: 'Initialization',
  usage: 'Usage',
  resources: 'Available Resources and Operations',
  errors: 'Error Handling',
  functions: 'Standalone Functions',
} as const

function header(note?: string): string {
  const lines = [
    '# OpenMeter SDK',
    '',
    'TypeScript client for the OpenMeter API — usage metering and billing for',
    'AI and DevTool companies. This package is generated from the OpenMeter',
    'TypeSpec definitions and ships fully-typed request and response models.',
  ]
  if (note) {
    lines.push('', note.trim())
  }
  return lines.join('\n')
}

function tableOfContents(resources: ReadmeResource[]): string {
  const lines = [`## ${HEADINGS.toc}`, '']
  lines.push(tocEntry(HEADINGS.install, 0))
  lines.push(tocEntry(HEADINGS.init, 0))
  lines.push(tocEntry(HEADINGS.usage, 0))
  lines.push(tocEntry(HEADINGS.resources, 0))
  for (const { resource } of resources) {
    const { class: cls } = namespaceNames(resource)
    lines.push(tocEntry(cls, 1))
  }
  lines.push(tocEntry(HEADINGS.errors, 0))
  lines.push(tocEntry(HEADINGS.functions, 0))
  return lines.join('\n')
}

function installation(packageName: string): string {
  return [
    `## ${HEADINGS.install}`,
    '',
    '```bash',
    `npm install ${packageName}`,
    '```',
    '',
    'Or with your package manager of choice:',
    '',
    '```bash',
    `pnpm add ${packageName}`,
    `yarn add ${packageName}`,
    '```',
  ].join('\n')
}

function initialization(packageName: string): string {
  return [
    `## ${HEADINGS.init}`,
    '',
    'Create a client with a base URL and an API key. The API key is sent as a',
    '`Bearer` token on every request.',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '```',
    '',
    'Konnect regions are addressed with a server template and a `region`',
    'variable:',
    '',
    '```typescript',
    `import { OpenMeter, ServerList } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    '  baseUrl: ServerList[0],',
    "  serverVariables: { region: 'eu' },",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '```',
    '',
    'The `apiKey` may also be a function returning a `string` or',
    '`Promise<string>`, so tokens can be refreshed per request.',
  ].join('\n')
}

function usage(packageName: string): string {
  return [
    `## ${HEADINGS.usage}`,
    '',
    'Every operation is reachable through a fluent, namespaced client and',
    'returns a typed response (or throws an `HTTPError` on a non-2xx status).',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'const meter = await client.meters.create({',
    "  name: 'Tokens',",
    "  key: 'tokens',",
    "  aggregation: 'sum',",
    "  eventType: 'request',",
    "  valueProperty: '$.tokens',",
    '})',
    '',
    'const meters = await client.meters.list()',
    '```',
    '',
    'Each method takes the request object as its first argument and an optional',
    'per-request options object (`RequestOptions`) as its second.',
  ].join('\n')
}

function resourcesSection(resources: ReadmeResource[]): string {
  const blocks = [
    `## ${HEADINGS.resources}`,
    '',
    'Operations are grouped by resource and exposed as methods on the client.',
    'The full call path, HTTP route, and a short description are listed below.',
  ]
  for (const { resource, ops } of resources) {
    const { class: cls, getter } = namespaceNames(resource)
    blocks.push('', `### ${cls}`, '', operationsTable(getter, ops))
  }
  return blocks.join('\n')
}

function errorHandling(packageName: string): string {
  return [
    `## ${HEADINGS.errors}`,
    '',
    'A non-2xx response rejects with an `HTTPError` carrying the problem-details',
    'fields (`status`, `type`, `title`, `url`) from the response.',
    '',
    '```typescript',
    `import { OpenMeter, HTTPError } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'try {',
    "  await client.meters.get({ meterId: 'unknown' })",
    '} catch (error) {',
    '  if (error instanceof HTTPError) {',
    '    console.error(error.status, error.title, error.type)',
    '  }',
    '}',
    '```',
  ].join('\n')
}

function standaloneFunctions(packageName: string): string {
  return [
    `## ${HEADINGS.functions}`,
    '',
    'Every method is also available as a standalone, tree-shakeable function',
    'that takes a `Client` and returns a `Result` instead of throwing.',
    '',
    '```typescript',
    `import { Client, funcs } from '${packageName}'`,
    '',
    'const client = new Client({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'const result = await funcs.listMeters(client)',
    'if (result.ok) {',
    '  console.log(result.value)',
    '} else {',
    '  console.error(result.error)',
    '}',
    '```',
  ].join('\n')
}

/** The package README, built from the same grouped operations the SDK files
 * are generated from, so the documented call paths and routes always match the
 * emitted client. */
export function readmeFile(
  resources: ReadmeResource[],
  packageName: string,
  note?: string,
): string {
  return (
    [
      header(note),
      tableOfContents(resources),
      installation(packageName),
      initialization(packageName),
      usage(packageName),
      resourcesSection(resources),
      errorHandling(packageName),
      standaloneFunctions(packageName),
    ].join('\n\n') + '\n'
  )
}
