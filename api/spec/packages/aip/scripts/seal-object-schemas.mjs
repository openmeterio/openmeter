#!/usr/bin/env node

import fs from 'node:fs/promises'
import process from 'node:process'
import YAML from 'yaml'

const YAML_OPTIONS = {
  indent: 2,
  lineWidth: 0,
}

function isPlainObject(value) {
  return value != null && typeof value === 'object' && !Array.isArray(value)
}

// `@typespec/openapi3` with `seal-object-schemas: true` emits
// `additionalProperties: { not: {} }` to forbid extra properties.
// kin-openapi's deepObject decoder cannot pick the matching branch of a
// `oneOf` when the branch carries that form, so every nested-object
// query (e.g. `filter[boolean][eq]=true`) becomes "path is not
// convertible to primitive". `additionalProperties: false` is
// semantically equivalent and accepted by kin-openapi, so rewrite it.
function isNotEmptyObject(value) {
  if (!isPlainObject(value)) {
    return false
  }

  const keys = Object.keys(value)
  if (keys.length !== 1 || keys[0] !== 'not') {
    return false
  }

  return isPlainObject(value.not) && Object.keys(value.not).length === 0
}

function rewriteAdditionalProperties(node) {
  if (Array.isArray(node)) {
    let changed = false

    for (const item of node) {
      changed = rewriteAdditionalProperties(item) || changed
    }

    return changed
  }

  if (!isPlainObject(node)) {
    return false
  }

  let changed = false

  if (isNotEmptyObject(node.additionalProperties)) {
    node.additionalProperties = false
    changed = true
  }

  for (const value of Object.values(node)) {
    changed = rewriteAdditionalProperties(value) || changed
  }

  return changed
}

async function pathExists(path) {
  try {
    await fs.access(path)
    return true
  } catch {
    return false
  }
}

async function validateInputFile(filePath) {
  if (!(await pathExists(filePath))) {
    throw new Error(`file not found: ${filePath}`)
  }

  const stat = await fs.stat(filePath)
  if (!stat.isFile()) {
    throw new Error(`not a file: ${filePath}`)
  }
}

async function readYamlFile(filePath) {
  const raw = await fs.readFile(filePath, 'utf8')

  try {
    return YAML.parse(raw)
  } catch (error) {
    throw new Error(`parse error: ${error.message}`)
  }
}

async function writeYamlFile(filePath, document) {
  const output = YAML.stringify(document, YAML_OPTIONS)
  const normalized = output.endsWith('\n') ? output : `${output}\n`

  await fs.writeFile(filePath, normalized, 'utf8')
}

function printUsage() {
  process.stderr.write(
    'Usage: seal-object-schemas.mjs <openapi.yaml-or-glob> [<openapi.yaml-or-glob> ...]\n',
  )
}

async function processFile(filePath) {
  await validateInputFile(filePath)

  const parsed = await readYamlFile(filePath)
  const changed = rewriteAdditionalProperties(parsed)

  if (changed) {
    await writeYamlFile(filePath, parsed)
  }
}

function looksLikeGlob(pattern) {
  return /[*?[]/.test(pattern)
}

async function expandPattern(pattern) {
  if (!looksLikeGlob(pattern)) {
    return [pattern]
  }

  const matches = []
  for await (const match of fs.glob(pattern)) {
    matches.push(match)
  }
  matches.sort()

  if (matches.length === 0) {
    throw new Error(`no files matched pattern: ${pattern}`)
  }

  return matches
}

async function main() {
  const patterns = process.argv.slice(2)

  if (patterns.length === 0) {
    printUsage()
    process.exitCode = 1
    return
  }

  try {
    const seen = new Set()
    for (const pattern of patterns) {
      const filePaths = await expandPattern(pattern)
      for (const filePath of filePaths) {
        if (seen.has(filePath)) {
          continue
        }
        seen.add(filePath)
        await processFile(filePath)
      }
    }
  } catch (error) {
    process.stderr.write(`seal-object-schemas: ${error.message}\n`)
    process.exitCode = 1
  }
}

await main()
