#!/usr/bin/env node

import fs from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import YAML from 'yaml'

const FLATTEN_MARKER = 'x-flatten-allOf'
const YAML_OPTIONS = { indent: 2, lineWidth: 0 }

/**
 * Whether the input is a plain, non-array object.
 */
function isPlainObject(value) {
  return value != null && typeof value === 'object' && !Array.isArray(value)
}

async function safeStat(p) {
  try {
    return await fs.stat(p)
  } catch {
    return null
  }
}

/**
 * Only touch generated v3 OpenAPI files.
 */
function shouldProcessFile(p) {
  return p.includes(`${path.sep}v3${path.sep}`) && p.endsWith('.yaml')
}

/**
 * Move sibling properties into an allOf branch when the schema contains a
 * referenced member. This keeps flattening hints consistent across the file.
 */
function flattenAllOf(node) {
  if (Array.isArray(node)) {
    return node.reduce((acc, item) => flattenAllOf(item) || acc, false)
  }

  if (!isPlainObject(node)) return false

  let changed = false
  const allOf = node.allOf

  const hasRefInAllOf =
    Array.isArray(allOf) &&
    allOf.some((item) => isPlainObject(item) && typeof item.$ref === 'string')

  if (hasRefInAllOf) {
    const movableKeys = Object.keys(node).filter(
      (key) =>
        key !== 'allOf' && key !== FLATTEN_MARKER && !key.startsWith('x-'),
    )

    if (movableKeys.length > 0) {
      const moved = {}
      for (const key of movableKeys) {
        moved[key] = node[key]
        delete node[key]
      }

      node.allOf.push(moved)
      node[FLATTEN_MARKER] = true
      changed = true
    } else if (node[FLATTEN_MARKER] !== true) {
      // Keep behavior consistent: if it's an allOf+$ref schema, mark it flattenable.
      node[FLATTEN_MARKER] = true
      changed = true
    }
  }

  for (const value of Object.values(node)) {
    changed = flattenAllOf(value) || changed
  }

  return changed
}

async function readYaml(filePath) {
  const raw = await fs.readFile(filePath, 'utf8')
  try {
    return YAML.parse(raw)
  } catch (error) {
    throw new Error(`Failed to parse YAML at ${filePath}: ${error.message}`)
  }
}

async function writeYaml(filePath, data) {
  const serialized = YAML.stringify(data, YAML_OPTIONS)
  const content = serialized.endsWith('\n') ? serialized : `${serialized}\n`
  await fs.writeFile(filePath, content, 'utf8')
}

async function processFile(filePath) {
  const parsed = await readYaml(filePath)
  const changed = flattenAllOf(parsed)

  if (changed) {
    await writeYaml(filePath, parsed)
  }

  return { filePath, changed }
}

async function collectFiles(args) {
  const files = []
  let hadError = false

  for (const arg of args) {
    const abs = path.resolve(arg)
    const stat = await safeStat(abs)

    if (!stat) {
      process.stderr.write(`flatten-allof-v3: file not found: ${arg}\n`)
      hadError = true
      continue
    }

    if (!stat.isFile()) {
      process.stderr.write(`flatten-allof-v3: not a file: ${arg}\n`)
      hadError = true
      continue
    }

    if (!shouldProcessFile(abs)) {
      process.stderr.write(`flatten-allof-v3: skipped (not v3): ${arg}\n`)
      continue
    }

    files.push(abs)
  }

  return { files: [...new Set(files)].sort(), hadError }
}

async function main() {
  const args = process.argv.slice(2)

  if (args.length === 0) {
    process.stderr.write(
      'Usage: flatten-allof-v3.mjs <path/to/openapi.yaml> [more.yaml ...]\n',
    )
    process.exitCode = 1
    return
  }

  const { files, hadError } = await collectFiles(args)

  let changedCount = 0
  for (const filePath of files) {
    const { changed } = await processFile(filePath)
    if (changed) changedCount++
  }

  if (hadError) {
    process.exitCode = 1
  }

  process.stdout.write(
    `flatten-allof-v3: processed ${files.length} file(s), changed ${changedCount} file(s)\n`,
  )
}

await main().catch((error) => {
  process.stderr.write(`flatten-allof-v3: unexpected error: ${error.message}\n`)
  process.exitCode = 1
})
