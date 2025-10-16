import fs from 'node:fs'
import path from 'node:path'
import { parseWithPointers, getLocationForJsonPath } from '@stoplight/yaml'

// Resolve repo root relative to this file (api/spec/scripts/ -> repo root)
const repoRoot = path.resolve(
  path.join(
    import.meta.url.startsWith('file:')
      ? new URL('.', import.meta.url).pathname
      : process.cwd(),
    '../../..',
  ),
)

const files = [
  path.join(repoRoot, 'api/openapi.yaml'),
  path.join(repoRoot, 'api/openapi.cloud.yaml'),
]

// Target JSON pointer and path
const TARGET_PTR =
  '/components/schemas/BillingWorkflowCollectionSettings/properties/alignment/default'
const TARGET_PATH = [
  'components',
  'schemas',
  'BillingWorkflowCollectionSettings',
  'properties',
  'alignment',
  'default',
] as const

// The exact YAML line to insert after the key's newline
const TARGET_VALUE_LINE = 'type: subscription'

// Convert (line, character) to absolute offset in the source text
function toOffset(lineMap: number[], line: number, ch: number): number {
  return (line === 0 ? 0 : lineMap[line - 1]) + ch
}

// Locate the 'default:' key line and return its indent
function findKeyIndent(
  src: string,
  valueStartOffset: number,
): { keyIndent: string; keyLineBegin: number } {
  const searchWindowStart = Math.max(0, valueStartOffset - 300)
  const window = src.slice(searchWindowStart, valueStartOffset)
  const defaultIdxInWindow = window.lastIndexOf('default:')
  const keyColonPos =
    defaultIdxInWindow !== -1
      ? searchWindowStart + defaultIdxInWindow + 'default:'.length - 1
      : valueStartOffset
  const keyLineStart = src.lastIndexOf('\n', keyColonPos)
  const keyLineBegin = keyLineStart === -1 ? 0 : keyLineStart + 1
  const indentMatch = src.slice(keyLineBegin, keyColonPos + 1).match(/^[ \t]*/)
  return { keyIndent: indentMatch ? indentMatch[0] : '', keyLineBegin }
}

// Infer one indent unit used under the key by scanning nearby lines
function inferIndentUnit(
  src: string,
  keyIndent: string,
  keyLineBegin: number,
  valueEndOffset: number,
): string {
  // Try scanning upwards to find a sibling with deeper indent
  let prevLineStart = src.lastIndexOf('\n', keyLineBegin - 2)
  while (prevLineStart >= 0) {
    const currBegin = prevLineStart + 1
    const currEnd = src.indexOf('\n', currBegin)
    const ln = src.slice(currBegin, currEnd === -1 ? src.length : currEnd)
    if (ln.trim().length > 0) {
      const m = ln.match(/^[ \t]*/)
      const ind = m ? m[0] : ''
      if (ind.startsWith(keyIndent) && ind.length > keyIndent.length) {
        return ind.slice(keyIndent.length)
      }
    }
    prevLineStart = src.lastIndexOf('\n', prevLineStart - 1)
  }

  // Fallback: scan downwards
  let cursor = valueEndOffset
  for (let i = 0; i < 8; i++) {
    const nl = src.indexOf('\n', cursor)
    if (nl === -1) break
    const nextNl = src.indexOf('\n', nl + 1)
    const line = src.slice(nl + 1, nextNl === -1 ? src.length : nextNl)
    if (line.trim().length === 0) {
      cursor = nl + 1
      continue
    }
    const m = line.match(/^[ \t]*/)
    const ind = m ? m[0] : ''
    if (ind.startsWith(keyIndent) && ind.length > keyIndent.length) {
      return ind.slice(keyIndent.length)
    }
    cursor = nl + 1
  }

  return '  '
}

// Find where to start replacing (after the colon), trimming any whitespace
function findAfterColonOffset(src: string, valueStartOffset: number): number {
  let j = valueStartOffset - 1
  while (
    j >= 0 &&
    (src[j] === ' ' || src[j] === '\t' || src[j] === '\n' || src[j] === '\r')
  )
    j--
  return src[j] === ':' ? j + 1 : valueStartOffset
}

// Remove any duplicate sibling "type: subscription" line that might follow insertion
function removeDuplicateSibling(
  updated: string,
  insertPos: number,
  keyIndent: string,
  replacement: string,
  newline: string,
): string {
  const dupLine = newline + keyIndent + TARGET_VALUE_LINE
  const afterInsert = insertPos + replacement.length
  if (updated.slice(afterInsert, afterInsert + dupLine.length) === dupLine) {
    return (
      updated.slice(0, afterInsert) +
      updated.slice(afterInsert + dupLine.length)
    )
  }
  return updated
}

function detectNewline(src: string): string {
  return src.includes('\r\n') ? '\r\n' : '\n'
}

function updateOne(filePath: string): boolean {
  const src = fs.readFileSync(filePath, 'utf8')
  const result = parseWithPointers(src, { ignoreDuplicateKeys: false }) as any
  const nodeLoc = getLocationForJsonPath(
    result,
    TARGET_PATH as unknown as (string | number)[],
  )
  if (!nodeLoc || !nodeLoc.range) {
    console.error(`[skip] Cannot locate value for ${TARGET_PTR} in ${filePath}`)
    return false
  }

  const valueStart = toOffset(
    result.lineMap,
    nodeLoc.range.start.line,
    nodeLoc.range.start.character,
  )
  const valueEnd = toOffset(
    result.lineMap,
    nodeLoc.range.end.line,
    nodeLoc.range.end.character,
  )

  const { keyIndent, keyLineBegin } = findKeyIndent(src, valueStart)
  const indentUnit = inferIndentUnit(src, keyIndent, keyLineBegin, valueEnd)
  const contentIndent = keyIndent + indentUnit
  const newline = detectNewline(src)
  const replacement = newline + contentIndent + TARGET_VALUE_LINE

  const wsStart = findAfterColonOffset(src, valueStart)
  const before = src.slice(0, wsStart)
  const after = src.slice(valueEnd)
  let updated = before + replacement + after
  updated = removeDuplicateSibling(
    updated,
    before.length,
    keyIndent,
    replacement,
    newline,
  )

  if (updated === src) {
    console.log(`[noop] ${filePath}`)
    return false
  }
  fs.writeFileSync(filePath, updated, 'utf8')
  console.log(`[ok] ${filePath}`)
  return true
}

let changed = false
for (const f of files) {
  changed = updateOne(f) || changed
}
process.exitCode = 0
