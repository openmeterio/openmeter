#!/usr/bin/env node

// Removes spurious blank lines that appear before chained `.method()` calls in
// the generated Zod models. The emitter produces these as a side effect of
// alloy's nested `formatCallChain` + `formatDotAccess` groups when a single
// chained call (typically `.describe(longString)`) overflows the print width;
// prettier preserves the blank line because it sits inside an expression
// chain. Matching `\n\n` followed by indented `.` is unambiguous for these
// schema files — no valid Zod expression begins a continuation chunk with a
// blank line in this generated output.

import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const [, , target] = process.argv
if (!target) {
  console.error('clean-zod-blank-lines: missing target file path')
  process.exit(1)
}

const file = path.resolve(process.cwd(), target)
const source = await readFile(file, 'utf8')
const cleaned = source.replace(/\n\n(\s+\.)/g, '\n$1')

if (cleaned !== source) {
  await writeFile(file, cleaned)
}
