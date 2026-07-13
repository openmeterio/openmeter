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
  // `@doc` is the longer description; operations that only carry a
  // `@summary` (no `@doc`) would otherwise render a blank Description cell.
  const text = op.doc ?? op.summary
  if (!text) {
    return ''
  }
  return text
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
  config: 'Configuration',
  usage: 'Usage',
  pagination: 'Pagination',
  resources: 'Available Resources and Operations',
  internal: 'Internal Operations',
  validate: 'Runtime Validation (validate option)',
  zod: 'Zod Schemas (./zod export)',
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

function tableOfContents(
  resources: ReadmeResource[],
  internalResources: ReadmeResource[],
): string {
  const lines = [`## ${HEADINGS.toc}`, '']
  lines.push(tocEntry(HEADINGS.install, 0))
  lines.push(tocEntry(HEADINGS.init, 0))
  lines.push(tocEntry(HEADINGS.config, 0))
  lines.push(tocEntry(HEADINGS.usage, 0))
  lines.push(tocEntry(HEADINGS.pagination, 0))
  lines.push(tocEntry(HEADINGS.resources, 0))
  for (const { resource } of resources) {
    const { class: cls } = namespaceNames(resource)
    lines.push(tocEntry(cls, 1))
  }
  if (internalResources.length > 0) {
    lines.push(tocEntry(HEADINGS.internal, 0))
    for (const { resource } of internalResources) {
      lines.push(tocEntry(internalResourceHeading(resource), 1))
    }
  }
  lines.push(tocEntry(HEADINGS.validate, 0))
  lines.push(tocEntry(HEADINGS.zod, 0))
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

function configuration(packageName: string): string {
  return [
    `## ${HEADINGS.config}`,
    '',
    "`SDKOptions` extends [ky](https://github.com/sindresorhus/ky)'s",
    '`Options`, so every transport setting ky supports is a top-level client',
    'option: retry policy, per-attempt timeout (ky defaults to 10 seconds),',
    'lifecycle hooks, a custom `fetch`, and so on.',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '  timeout: 30_000,',
    '  hooks: {',
    '    beforeRequest: [',
    '      ({ request }) => {',
    '        console.log(`-> ${request.method} ${request.url}`)',
    '      },',
    '    ],',
    '  },',
    '  fetch: async (input, init) => {',
    '    const start = Date.now()',
    '    const response = await fetch(input, init)',
    '    console.log(`${response.status} in ${Date.now() - start}ms`)',
    '    return response',
    '  },',
    '})',
    '```',
    '',
    'ky only retries the idempotent methods by default — `get`, `put`,',
    '`head`, `delete`, `options`, `trace` — never `post`. That means a',
    'dropped `client.events.ingest` call is not retried on a network error',
    'or a 5xx response unless you opt in explicitly:',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    "  retry: { limit: 3, methods: ['get', 'put', 'head', 'delete', 'post'] },",
    '})',
    '```',
    '',
    'This is safe specifically for event ingestion: the event `id` is its',
    'deduplication key server-side, so resending the same event on retry is',
    'a no-op rather than a duplicate.',
    '',
    'Every method also takes a per-request `RequestOptions` as its second',
    "argument — a curated subset of ky's options (`signal`, `headers`,",
    '`timeout`, `retry`) applied to that call only:',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'const controller = new AbortController()',
    'setTimeout(() => controller.abort(), 5_000)',
    '',
    'const meters = await client.meters.list(undefined, {',
    '  signal: controller.signal,',
    "  headers: { 'X-Request-Id': 'batch-42' },",
    '})',
    '```',
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
    '',
    'Responses return date-time fields as native `Date` objects (every',
    '`createdAt`/`updatedAt`, meter query row windows, …), and requests accept',
    'either a `Date` or an RFC 3339 string — a meter query `from`/`to`, an',
    "ingested event's `time`, filter operands, all alike.",
  ].join('\n')
}

function pagination(packageName: string): string {
  return [
    `## ${HEADINGS.pagination}`,
    '',
    'Every list operation that returns pages also has an `…All` companion —',
    '`client.meters.listAll(request?)` alongside `client.meters.list(request?)` —',
    'that returns an `AsyncIterable` of items instead of one page. It fetches',
    'each following page lazily, only when the previous page is exhausted, so a',
    '`break` (or a `return`) partway through never fires a request for a page',
    'nothing consumes:',
    '',
    '```typescript',
    `import { OpenMeter } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'for await (const meter of client.meters.listAll()) {',
    '  console.log(meter.key)',
    '}',
    '```',
    '',
    'The request object accepts the same filters, sort, and page size as the',
    'single-page method — only the page cursor/number itself advances between',
    'requests:',
    '',
    '```typescript',
    "for await (const meter of client.meters.listAll({ filter: { key: 'api' } })) {",
    '  if (meter.key === "api-requests") {',
    '    break // stops iterating; no further pages are fetched',
    '  }',
    '}',
    '```',
    '',
    'Cursor-paginated resources (`events`, credit transactions) work the same',
    'way:',
    '',
    '```typescript',
    'for await (const event of client.events.listAll()) {',
    '  console.log(event.event.id)',
    '}',
    '```',
    '',
    'Auto-pagination takes the same optional `RequestOptions` as every other',
    'method, so one `AbortSignal` cancels the whole iteration, not just the page',
    'in flight:',
    '',
    '```typescript',
    'const controller = new AbortController()',
    'setTimeout(() => controller.abort(), 30_000)',
    '',
    'for await (const meter of client.meters.listAll(undefined, {',
    '  signal: controller.signal,',
    '})) {',
    '  console.log(meter.key)',
    '}',
    '```',
    '',
    'Iteration stops after the server’s last page — an empty or short-of-`size`',
    'page for a page-number resource, or a response with no next cursor for a',
    'cursor resource. A misbehaving server that never signals the end of the',
    'list fails fast instead of looping forever: after 10,000 pages the',
    'iterable throws `PaginationLimitExceededError`.',
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

/** Sub-heading for an internal resource group. Prefixed so it never collides
 * with the same resource's public section anchor. */
function internalResourceHeading(resource: string): string {
  return `Internal ${namespaceNames(resource).class}`
}

function internalSection(resources: ReadmeResource[]): string {
  const blocks = [
    `## ${HEADINGS.internal}`,
    '',
    'Operations marked internal in the API definition are exposed under',
    '`client.internal.*`, quarantined from the customer surface. They are not',
    'intended for customer use: they may require additional permissions, and',
    'they can change or be removed without notice or semver consideration.',
  ]
  for (const { resource, ops } of resources) {
    const { getter } = namespaceNames(resource)
    blocks.push(
      '',
      `### ${internalResourceHeading(resource)}`,
      '',
      operationsTable(`internal.${getter}`, ops),
    )
  }
  return blocks.join('\n')
}

function runtimeValidation(packageName: string): string {
  return [
    `## ${HEADINGS.validate}`,
    '',
    "`validate` is off by default. The SDK's normal request/response mapping",
    'only renames keys and converts dates — it never rejects a payload — so',
    'an additive server-side change (a new response field, a new enum value)',
    'never breaks a client running an older SDK version.',
    '',
    'Set `validate: true` to additionally check the wire payload — the',
    'request body after mapping to snake_case, and the raw response before',
    'mapping back — against the generated `…Wire` schemas. Those schemas are',
    'strict: an unknown field or an unrecognized enum value fails validation',
    'instead of being silently accepted. That makes `validate` a tool for',
    'catching SDK/server contract drift in development or CI, not something',
    'to run in production, where forward compatibility with additive server',
    'changes matters more than strict rejection.',
    '',
    '```typescript',
    `import { OpenMeter, ValidationError } from '${packageName}'`,
    '',
    'const client = new OpenMeter({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '  validate: true,',
    '})',
    '',
    'try {',
    "  await client.meters.get({ meterId: 'meter_123' })",
    '} catch (error) {',
    '  if (error instanceof ValidationError) {',
    '    console.error(error.issues)',
    '  }',
    '}',
    '```',
    '',
    'The standalone functions surface the same failure as a `Result` instead',
    'of throwing — check `result.error instanceof ValidationError`.',
  ].join('\n')
}

function zodSchemas(packageName: string): string {
  return [
    `## ${HEADINGS.zod}`,
    '',
    `\`${packageName}/zod\` exports every model twice: once shaped like the`,
    "SDK's public surface (camelCase keys, `Date` for date-time fields — the",
    'same shape `meter.eventType`/`meter.createdAt` have in TypeScript) and',
    'once as a strict `…Wire` schema shaped like the literal JSON the server',
    'sends and accepts (snake_case keys, RFC 3339 date-time strings, unknown',
    'fields rejected) — the same `…Wire` schemas the `validate` option checks',
    'internally.',
    '',
    'Use the public schemas to validate a payload the SDK did not produce —',
    'a webhook body, a cached record, a test fixture — before trusting its',
    'shape:',
    '',
    '```typescript',
    `import * as schemas from '${packageName}/zod'`,
    '',
    'const parsed = schemas.meter.safeParse({',
    "  id: '01HZY3W6VXQK6H3NPC6DFA0PJT',",
    "  name: 'Tokens',",
    "  key: 'tokens',",
    "  aggregation: 'sum',",
    "  eventType: 'request',",
    '  createdAt: new Date(),',
    '  updatedAt: new Date(),',
    '})',
    '',
    'if (parsed.success) {',
    '  console.log(parsed.data.eventType)',
    '}',
    '```',
  ].join('\n')
}

function errorHandling(packageName: string): string {
  return [
    `## ${HEADINGS.errors}`,
    '',
    "A non-2xx response rejects with an `HTTPError` (`error.name === 'HTTPError'`)",
    'carrying the problem-details fields (`status`, `type`, `title`, `url`)',
    'from the response.',
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
    '',
    '`error.retryAfter` is the delta-seconds form of a numeric `Retry-After`',
    'header (the common case on 429 and 503 responses) and `undefined`',
    'otherwise. A 400 Bad Request additionally carries a typed',
    '`error.invalidParameters` array describing which fields failed',
    'validation; `error.getField(key)` is an untyped escape hatch for any',
    'other problem-details extension member:',
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
    '  await client.meters.create({',
    "    name: 'Tokens',",
    "    key: 'tokens',",
    "    aggregation: 'sum',",
    "    eventType: 'request',",
    '  })',
    '} catch (error) {',
    '  if (error instanceof HTTPError && error.status === 400) {',
    '    for (const param of error.invalidParameters ?? []) {',
    '      console.error(param)',
    '    }',
    '  }',
    '}',
    '```',
    '',
    "The SDK's other typed errors are `ValidationError` (see",
    `[${HEADINGS.validate}](#${slug(HEADINGS.validate)})), \`UnsafeIntegerError\``,
    '(an `int64`/`uint64` value exceeds what JSON can represent without',
    'precision loss), `DepthLimitExceededError` (response data nested deeper',
    "than the mapper's safety limit), and `PaginationLimitExceededError` (an",
    `[${HEADINGS.pagination}](#${slug(HEADINGS.pagination)}) iterable exceeded`,
    'its page-count safety limit) — each distinguished the same way, by',
    '`instanceof` or by `.name`.',
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
    '',
    '`ok`, `err`, and `unwrap` — the helpers `Result` is built from — are',
    'exported too, so a func call can be unwrapped back into a throwing call',
    'where that is more convenient:',
    '',
    '```typescript',
    `import { Client, funcs, unwrap } from '${packageName}'`,
    '',
    'const client = new Client({',
    "  baseUrl: 'https://openmeter.cloud/api/v3',",
    '  apiKey: process.env.OPENMETER_API_KEY,',
    '})',
    '',
    'const meters = unwrap(await funcs.listMeters(client))',
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
  internalResources: ReadmeResource[] = [],
): string {
  return (
    [
      header(note),
      tableOfContents(resources, internalResources),
      installation(packageName),
      initialization(packageName),
      configuration(packageName),
      usage(packageName),
      pagination(packageName),
      resourcesSection(resources),
      ...(internalResources.length > 0
        ? [internalSection(internalResources)]
        : []),
      runtimeValidation(packageName),
      zodSchemas(packageName),
      errorHandling(packageName),
      standaloneFunctions(packageName),
    ].join('\n\n') + '\n'
  )
}
