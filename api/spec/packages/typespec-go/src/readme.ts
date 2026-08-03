import type { Operation, Program, Type } from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
import { goExportedName } from './go-types.js'
import { namespaceNames } from './grouping.js'
import { describeOperations, type GoOperation } from './operations.js'

export interface ReadmeService {
  root: string
  nestPath: string[]
  operations: Operation[]
}

interface ResourceSection {
  root: string
  operations: Array<{ service: ReadmeService; operation: GoOperation }>
}

const HEADINGS = {
  toc: 'Table of Contents',
  install: 'Installation',
  init: 'Initialization',
  usage: 'Usage',
  resources: 'Available Resources and Operations',
  errors: 'Error Handling',
  pagination: 'Pagination and Streaming',
} as const

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

function header(note?: string): string {
  const lines = [
    '# OpenMeter Go SDK',
    '',
    'Go client for the OpenMeter API — usage metering and billing for',
    'AI and DevTool companies. This package is generated from the OpenMeter',
    'TypeSpec definitions and ships typed request and response models.',
  ]
  if (note) {
    lines.push('', note.trim())
  }
  return lines.join('\n')
}

function tableOfContents(resources: ResourceSection[]): string {
  const lines = [`## ${HEADINGS.toc}`, '']
  lines.push(tocEntry(HEADINGS.install, 0))
  lines.push(tocEntry(HEADINGS.init, 0))
  lines.push(tocEntry(HEADINGS.usage, 0))
  lines.push(tocEntry(HEADINGS.resources, 0))
  for (const resource of resources) {
    lines.push(tocEntry(namespaceNames(resource.root).class, 1))
  }
  lines.push(tocEntry(HEADINGS.errors, 0))
  lines.push(tocEntry(HEADINGS.pagination, 0))
  return lines.join('\n')
}

function installation(modulePath: string): string {
  return [
    `## ${HEADINGS.install}`,
    '',
    '```bash',
    `go get ${modulePath}`,
    '```',
  ].join('\n')
}

function initialization(modulePath: string, packageName: string): string {
  return [
    `## ${HEADINGS.init}`,
    '',
    'Create a client with a base URL and an API key. The API key is sent as a',
    '`Bearer` token on every request.',
    '',
    '```go',
    'package main',
    '',
    'import (',
    '\t"log"',
    '\t"os"',
    '',
    `\t"${modulePath}"`,
    ')',
    '',
    'func main() {',
    `\tom, err := ${packageName}.New(`,
    '\t\t"https://openmeter.cloud/api/v3",',
    `\t\t${packageName}.WithToken(os.Getenv("OPENMETER_API_KEY")),`,
    '\t)',
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '',
    '\t_ = om',
    '}',
    '```',
    '',
    'For region-specific deployments, pass the concrete API base URL for that',
    'region to `New`.',
  ].join('\n')
}

function usage(modulePath: string, packageName: string): string {
  return [
    `## ${HEADINGS.usage}`,
    '',
    'Every operation is reachable through a namespaced service on the client and',
    'returns a typed response plus an `error`.',
    '',
    '```go',
    'package main',
    '',
    'import (',
    '\t"context"',
    '\t"log"',
    '\t"os"',
    '',
    `\t"${modulePath}"`,
    ')',
    '',
    'func main() {',
    `\tom, err := ${packageName}.New(`,
    '\t\t"https://openmeter.cloud/api/v3",',
    `\t\t${packageName}.WithToken(os.Getenv("OPENMETER_API_KEY")),`,
    '\t)',
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '',
    '\tctx := context.Background()',
    `\tmeter, err := om.Meters.Create(ctx, ${packageName}.CreateMeterRequest{`,
    '\t\tName:          "Tokens",',
    '\t\tKey:           "tokens",',
    `\t\tAggregation:   ${packageName}.MeterAggregationSum,`,
    '\t\tEventType:     "request",',
    `\t\tValueProperty: ${packageName}.String("$.tokens"),`,
    '\t})',
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '',
    `\tmeters, err := om.Meters.List(ctx, ${packageName}.MeterListParams{})`,
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '',
    '\t_, _ = meter, meters',
    '}',
    '```',
    '',
    'Operation arguments follow the generated method signature: path parameters',
    'come first, then a typed request body when present, then typed query params',
    'when present.',
  ].join('\n')
}

function callPath(service: ReadmeService, methodName: string): string {
  return ['om', service.root, ...service.nestPath, methodName].join('.')
}

// Mirrors GoResource's isTextResponse: text responses grow a Stream method
// variant alongside the byte-returning method.
function isTextResponse(operation: GoOperation): boolean {
  return operation.responseContentType?.startsWith('text/') ?? false
}

function summaryCell(program: Program, op: GoOperation): string {
  const summary = $(program).type.getDoc(op.operation)
  if (!summary) {
    return ''
  }

  return summary
    .trim()
    .replace(/\s+/g, ' ')
    .replace(/\\/g, '\\\\')
    .replace(/\|/g, '\\|')
}

function operationsTable(
  program: Program,
  operations: ResourceSection['operations'],
): string {
  const header = ['| Method | HTTP | Description |', '| --- | --- | --- |']
  const rows = operations.flatMap(({ service, operation }) => {
    const call = `\`${callPath(service, operation.methodName)}\``
    const http = `\`${operation.verb.toUpperCase()} ${operation.path}\``
    const row = `| ${call} | ${http} | ${summaryCell(program, operation)} |`
    if (!isTextResponse(operation)) {
      return [row]
    }
    const streamName = `${goExportedName(operation.methodName)}Stream`
    const streamCall = `\`${callPath(service, streamName)}\``
    return [
      row,
      `| ${streamCall} | ${http} | Streaming variant of \`${operation.methodName}\` returning an \`io.ReadCloser\`. |`,
    ]
  })
  return [...header, ...rows].join('\n')
}

function resourcesSection(
  program: Program,
  resources: ResourceSection[],
): string {
  const blocks = [
    `## ${HEADINGS.resources}`,
    '',
    'Operations are grouped by resource and exposed as services on the client.',
    'The full call path, HTTP route, and a short description are listed below.',
  ]

  for (const resource of resources) {
    blocks.push(
      '',
      `### ${namespaceNames(resource.root).class}`,
      '',
      operationsTable(program, resource.operations),
    )
  }

  return blocks.join('\n')
}

function errorHandling(modulePath: string, packageName: string): string {
  return [
    `## ${HEADINGS.errors}`,
    '',
    'A non-2xx response returns an `*APIError` carrying the problem-details',
    'fields (`StatusCode`, `Status`, `Type`, `Title`, `Detail`, `Instance`) from',
    'the response where available. Client-side validation errors such as an empty',
    'path ID are returned before any HTTP request is made.',
    '',
    '```go',
    'package main',
    '',
    'import (',
    '\t"context"',
    '\t"errors"',
    '\t"log"',
    '',
    `\t"${modulePath}"`,
    ')',
    '',
    'func main() {',
    `\tom, err := ${packageName}.New("https://openmeter.cloud/api/v3", ${packageName}.WithToken("om_..."))`,
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '',
    '\t_, err = om.Meters.Get(context.Background(), "unknown")',
    '\tif err != nil {',
    `\t\tvar apiErr *${packageName}.APIError`,
    '\t\tif errors.As(err, &apiErr) {',
    '\t\t\tlog.Printf("%d %s %s", apiErr.StatusCode, apiErr.Title, apiErr.Type)',
    '\t\t\treturn',
    '\t\t}',
    '\t\tlog.Fatal(err)',
    '\t}',
    '}',
    '```',
  ].join('\n')
}

function paginationAndStreaming(packageName: string): string {
  return [
    `## ${HEADINGS.pagination}`,
    '',
    'Paginated list operations also emit `ListAll` helpers that return',
    '`iter.Seq2[T, error]`. Text responses such as meter CSV export emit a byte',
    'returning method and a `Stream` variant for callers that want an',
    '`io.ReadCloser`.',
    '',
    'Cursor-paginated responses report their position as `Next` and `Previous`',
    'on `CursorMetaPage`. Both are opaque cursor tokens: pass them back verbatim',
    'as the `page[after]` / `page[before]` query parameters',
    '(`CursorPageParams.After` / `CursorPageParams.Before`); do not parse or',
    'construct them.',
    '',
    'Iterating with `Before` set walks pages backward while the items within',
    'each page stay in forward order, so the resulting stream is not globally',
    'sorted.',
    '',
    '```go',
    `for meter, err := range om.Meters.ListAll(ctx, ${packageName}.MeterListParams{}) {`,
    '\tif err != nil {',
    '\t\tlog.Fatal(err)',
    '\t}',
    '\tlog.Println(meter.Key)',
    '}',
    '',
    `stream, err := om.Meters.QueryCSVStream(ctx, "meter-id", ${packageName}.MeterQueryRequest{})`,
    'if err != nil {',
    '\tlog.Fatal(err)',
    '}',
    'defer stream.Close()',
    '```',
  ].join('\n')
}

function resourceSections(
  program: Program,
  services: ReadmeService[],
  bodyOverrides: Map<string, Type>,
): ResourceSection[] {
  const byRoot = new Map<string, ResourceSection>()
  for (const service of services) {
    const section = byRoot.get(service.root) ?? {
      root: service.root,
      operations: [],
    }
    if (!byRoot.has(service.root)) {
      byRoot.set(service.root, section)
    }

    for (const operation of describeOperations(
      program,
      service.root,
      service.operations,
      bodyOverrides,
      service.nestPath,
    )) {
      section.operations.push({ service, operation })
    }
  }

  return [...byRoot.values()].filter(
    (resource) => resource.operations.length > 0,
  )
}

/** The package README, built from the same grouped operations the Go SDK files
 * are generated from, so the documented call paths and routes always match the
 * emitted client. */
export function readmeFile(
  program: Program,
  modulePath: string,
  packageName: string,
  services: ReadmeService[],
  bodyOverrides: Map<string, Type>,
  note?: string,
): string {
  const resources = resourceSections(program, services, bodyOverrides)

  return (
    [
      header(note),
      tableOfContents(resources),
      installation(modulePath),
      initialization(modulePath, packageName),
      usage(modulePath, packageName),
      resourcesSection(program, resources),
      errorHandling(modulePath, packageName),
      paginationAndStreaming(packageName),
    ].join('\n\n') + '\n'
  )
}
