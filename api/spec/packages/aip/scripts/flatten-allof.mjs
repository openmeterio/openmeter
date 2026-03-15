#!/usr/bin/env node

import fs from 'node:fs/promises'
import process from 'node:process'
import YAML from 'yaml'

const FLATTEN_MARKER = 'x-flatten-allOf'
const YAML_OPTIONS = {
  indent: 2,
  lineWidth: 0,
}

function isPlainObject(value) {
  return value != null && typeof value === 'object' && !Array.isArray(value)
}

function isRefObject(value) {
  return isPlainObject(value) && typeof value.$ref === 'string'
}

function getMovableKeys(node) {
  return Object.keys(node).filter(
    (key) => key !== 'allOf' && key !== FLATTEN_MARKER && !key.startsWith('x-'),
  )
}

function moveSiblingPropertiesIntoAllOf(node) {
  const movableKeys = getMovableKeys(node)

  if (movableKeys.length === 0) {
    if (node[FLATTEN_MARKER] !== true) {
      node[FLATTEN_MARKER] = true
      return true
    }

    return false
  }

  const moved = {}
  for (const key of movableKeys) {
    moved[key] = node[key]
    delete node[key]
  }

  node.allOf.push(moved)
  node[FLATTEN_MARKER] = true

  return true
}

/**
 * Move sibling properties into an allOf branch when the schema contains a
 * referenced member. This keeps flattening hints consistent across the file.
 */
function flattenAllOf(node) {
  if (Array.isArray(node)) {
    let changed = false

    for (const item of node) {
      changed = flattenAllOf(item) || changed
    }

    return changed
  }

  if (!isPlainObject(node)) {
    return false
  }

  let changed = false
  const { allOf } = node

  const hasRefInAllOf =
    Array.isArray(allOf) && allOf.some((item) => isRefObject(item))

  if (hasRefInAllOf) {
    changed = moveSiblingPropertiesIntoAllOf(node) || changed
  }

  for (const value of Object.values(node)) {
    changed = flattenAllOf(value) || changed
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
  process.stderr.write('Usage: flatten-allof.mjs <openapi.yaml>\n')
}

async function main() {
  const [filePath] = process.argv.slice(2)

  if (!filePath) {
    printUsage()
    process.exitCode = 1
    return
  }

  try {
    await validateInputFile(filePath)

    const parsed = await readYamlFile(filePath)
    const changed = flattenAllOf(parsed)

    if (changed) {
      await writeYamlFile(filePath, parsed)
    }
  } catch (error) {
    process.stderr.write(`flatten-allof: ${error.message}\n`)
    process.exitCode = 1
  }
}

await main()
