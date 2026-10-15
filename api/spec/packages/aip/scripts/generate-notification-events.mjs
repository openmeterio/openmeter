// Generates the per-domain Notification Event artifacts and copies them into
// the Go module.
//
// Run after `tsp compile --config tspconfig.notification-events.yaml`, which
// emits the JSON Schemas, the generated Go files and manifest.json. For every @goPackage
// namespace in the manifest this script:
//   1. compiles a generated entry point that makes the namespace an OpenAPI
//      service, producing the events-only OpenAPI document of its types;
//   2. copies that document to its committed location under api/, and the
//      emitted generate.gen.go, eventtypes.gen.go and JSON Schemas into the Go
//      package directory. `go generate` then runs oapi-codegen through the
//      directive in generate.gen.go, like every other oapi-codegen target.
import { execFileSync } from 'node:child_process'
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const pkgRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = resolve(pkgRoot, '../../../..')
const out = join(pkgRoot, 'output/notification-events')
const manifest = JSON.parse(
  readFileSync(join(out, '_go/manifest.json'), 'utf8'),
)

for (const entry of manifest) {
  const entryDir = join(out, 'entries')
  const entryFile = join(entryDir, `${entry.dir.replaceAll('/', '_')}.tsp`)
  const openapiDir = join(out, 'openapi', entry.dir)
  mkdirSync(entryDir, { recursive: true })
  writeFileSync(
    entryFile,
    [
      `import "${relative(entryDir, join(pkgRoot, 'src/notification-events.tsp'))}";`,
      '',
      `@@TypeSpec.service(${entry.namespace}, #{ title: "${entry.namespace}" });`,
      '',
    ].join('\n'),
  )
  execFileSync(
    'tsp',
    [
      'compile',
      '--config',
      join(pkgRoot, 'tspconfig.notification-events-openapi.yaml'),
      entryFile,
      '--option',
      `@typespec/openapi3.emitter-output-dir=${openapiDir}`,
    ],
    { cwd: pkgRoot, stdio: 'inherit' },
  )
  // Same post-processing as v3: oapi-codegen and kin-openapi need
  // additionalProperties: false instead of the emitter's { not: {} }.
  execFileSync(
    'node',
    [
      join(pkgRoot, 'scripts/seal-object-schemas.mjs'),
      join(openapiDir, 'openapi.yaml'),
    ],
    {
      stdio: 'inherit',
    },
  )

  const goDir = join(repoRoot, entry.dir)
  mkdirSync(goDir, { recursive: true })
  mkdirSync(dirname(join(repoRoot, entry.spec)), { recursive: true })
  cpSync(join(openapiDir, 'openapi.yaml'), join(repoRoot, entry.spec))
  cpSync(
    join(out, '_go', entry.dir, 'generate.gen.go'),
    join(goDir, 'generate.gen.go'),
  )
  rmSync(join(goDir, 'eventtypes.gen.go'), { force: true })
  rmSync(join(goDir, 'schemas'), { recursive: true, force: true })
  if (entry.schemas.length > 0) {
    cpSync(
      join(out, '_go', entry.dir, 'eventtypes.gen.go'),
      join(goDir, 'eventtypes.gen.go'),
    )
    mkdirSync(join(goDir, 'schemas'))
    for (const { source, target } of entry.schemas) {
      cpSync(join(out, 'jsonschema', source), join(goDir, 'schemas', target))
    }
  }
}
