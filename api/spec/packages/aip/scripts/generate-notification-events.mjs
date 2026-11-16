// Copies the Notification Event definitions into the Go module.
//
// Run after `tsp compile --config tspconfig.yaml`, which emits the JSON
// Schemas, definitions.gen.go and manifest.json. The payload and Rule
// configuration Go types come from api/v3/models instead, because the v3 API
// references them.
import { cpSync, mkdirSync, readFileSync, rmSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const pkgRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = resolve(pkgRoot, '../../../..')
const out = join(pkgRoot, 'output/notification-events')
const manifest = JSON.parse(
  readFileSync(join(out, '_go/manifest.json'), 'utf8'),
)

const definitionsDir = join(repoRoot, manifest.definitions)
mkdirSync(definitionsDir, { recursive: true })
cpSync(
  join(out, '_go', manifest.definitions, 'definitions.gen.go'),
  join(definitionsDir, 'definitions.gen.go'),
)
rmSync(join(definitionsDir, 'schemas'), { recursive: true, force: true })
mkdirSync(join(definitionsDir, 'schemas'))
for (const { source, target } of manifest.schemas) {
  cpSync(
    join(out, 'jsonschema', source),
    join(definitionsDir, 'schemas', target),
  )
}
