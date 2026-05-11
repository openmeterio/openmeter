#!/usr/bin/env node

import { readFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { applyCodeFixes, compile, NodeHost } from '@typespec/compiler'

const CODEFIX_ID = 'format-doc-comment'
const RULE_NAME = 'doc-format'

const cwd = process.cwd()
const pkg = JSON.parse(
  await readFile(path.resolve(cwd, 'package.json'), 'utf8'),
)

// Derive the linter ruleset id from `package.json#name` (e.g.
// `@openmeter/api-spec-aip/all`) so renaming the package doesn't break this.
const RULE_ID = `${pkg.name}/${RULE_NAME}`
const RULESET = `${pkg.name}/all`

// Derive the TypeSpec entrypoint from `package.json#exports['.'].typespec`,
// matching how `tsp compile ./src` resolves it.
const entryRel = pkg.exports?.['.']?.typespec
if (!entryRel) {
  console.error(
    "apply-doc-fixes: package.json must declare exports['.'].typespec",
  )
  process.exit(1)
}
const entry = path.resolve(cwd, entryRel)

// `compile()` doesn't read `tspconfig.yaml`, so the linter ruleset has to be
// passed explicitly here. Mirror what `tspconfig.yaml` configures.
const program = await compile(NodeHost, entry, {
  noEmit: true,
  linterRuleSet: { extends: [RULESET] },
})

// Each lint diagnostic carries our `format-doc-comment` codefix plus an
// auto-attached `suppress` codefix from the compiler. Apply only ours.
const matching = program.diagnostics.filter((d) => d.code === RULE_ID)
const fixes = matching.flatMap(
  (d) => d.codefixes?.filter((c) => c.id === CODEFIX_ID) ?? [],
)

if (fixes.length === 0) {
  console.error('apply-doc-fixes: no doc-format codefixes to apply')
  process.exit(0)
}

await applyCodeFixes(NodeHost, fixes)

console.error(
  `apply-doc-fixes: applied ${fixes.length} fix(es) across ${matching.length} diagnostic(s)`,
)
