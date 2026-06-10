import * as ay from '@alloy-js/core'
import * as ts from '@alloy-js/typescript'
import {
  type EmitContext,
  getFriendlyName,
  ListenerFlow,
  type Model,
  navigateProgram,
  type Program,
  type Type,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
// Registers the experimental HTTP typekit ($.httpOperation, $.modelProperty
// HTTP helpers) used by the operation walk and metadata stripping.
import '@typespec/http/experimental/typekit'
import { Output, writeOutput } from '@typespec/emitter-framework'
import { ZodSchemaDeclaration } from './components/ZodSchemaDeclaration.jsx'
import { zod } from './external-packages/zod.js'
import type { ZodEmitterOptions } from './lib.js'
import { collectHttpOperations, operationSchemas } from './ZodOperations.jsx'
import { resolveStrippedNames } from './strip-prefixes.js'
import { newTopologicalTypeCollector } from './utils.jsx'
import { RUNTIME_TEMPLATES } from './runtime-templates.js'
import { interfacesFile } from './interface-types.js'
import { inputVariantName } from './input-variants.js'
import {
  groupOperations,
  jsonBodyOverrides,
  sdkOperation,
} from './sdk-operations.js'
import { requestTypesFor } from './request-types.js'
import { readmeFile, type ReadmeResource } from './readme.js'
import {
  facadeFile,
  funcsFile,
  funcsIndexFile,
  indexFile,
  namespaceFile,
  operationsAssertFile,
  operationsFile,
  sdkRootFile,
} from './sdk-files.js'

/**
 * The base name a type is declared under before prefix stripping: its
 * `@friendlyName` if present, otherwise its own name.
 */
function baseName(program: Program, type: Type): string | undefined {
  const friendly = getFriendlyName(program, type)
  if (friendly) return friendly
  return 'name' in type && typeof type.name === 'string' ? type.name : undefined
}

export async function $onEmit(context: EmitContext<ZodEmitterOptions>) {
  const types = getAllDataTypes(context.program)
  const tsNamePolicy = ts.createTSNamePolicy()
  const stripPrefixes = context.options['strip-name-prefixes'] ?? []

  const operations = collectHttpOperations(
    context.program,
    context.options['include-services'],
  )
  const opSchemas = operations.flatMap((op) =>
    operationSchemas(context.program, op),
  )

  // Pre-pass: collision-guarded resolved names so a strip is only applied when
  // it does not clash with another schema (mirrors the TS SDK emitter). The
  // per-operation schema names join the pool so strips never collide with them
  // either.
  const baseNames = [
    ...types
      .map((type) => baseName(context.program, type))
      .filter((n): n is string => n !== undefined),
    ...opSchemas.map((s) => s.baseName),
  ]
  const resolved = resolveStrippedNames(baseNames, stripPrefixes)

  const resolveName = (type: Type): string | undefined => {
    const base = baseName(context.program, type)
    return base ? (resolved.get(base) ?? base) : undefined
  }
  const models = types.filter((t): t is Model => t.kind === 'Model')
  const interfaceName = (name: string) =>
    tsNamePolicy.getName(name, 'interface')
  const interfaces = interfacesFile(
    context.program,
    models,
    resolveName,
    (name) => tsNamePolicy.getName(name, 'variable'),
    interfaceName,
  )

  // A type resolves to its documented interface only when its resolved name
  // matches an emitted interface (string-based, so template instantiations whose
  // Type identity differs from the collected model still resolve).
  const emittedInterfaceNames = new Set(
    models
      .map((m) => resolveName(m))
      .filter((n): n is string => Boolean(n))
      .map(interfaceName),
  )
  const resolveInterface = (type: Type | undefined): string | undefined => {
    if (!type) {
      return undefined
    }
    const resolved = resolveName(type)
    if (!resolved) {
      return undefined
    }
    const name = interfaceName(resolved)
    return emittedInterfaceNames.has(name) ? name : undefined
  }

  // String-keyed set of interface names whose input shape diverges, so a request
  // body resolves to its `…Input` variant even when the HTTP body Type identity
  // differs from the collected model (same bridging as `resolveInterface`).
  const divergentInterfaceNames = new Set(
    [...interfaces.divergentModels]
      .map((m) => resolveName(m))
      .filter((n): n is string => Boolean(n))
      .map(interfaceName),
  )
  const resolveRequestBody = (type: Type | undefined): string | undefined => {
    const name = resolveInterface(type)
    if (!name) {
      return undefined
    }
    return divergentInterfaceNames.has(name) ? inputVariantName(name) : name
  }

  const groups = groupOperations(operations)
  const resources = [...groups.keys()]
  const bodyOverrides = jsonBodyOverrides(context.program)
  const sdkFiles: Array<{ path: string; content: string }> = []
  const readmeResources: ReadmeResource[] = []
  for (const [resource, ops] of groups) {
    const sdkOps = ops.map((op) =>
      sdkOperation(
        context.program,
        op,
        resource,
        resolveInterface,
        resolveRequestBody,
        bodyOverrides,
      ),
    )
    readmeResources.push({ resource, ops: sdkOps })
    const file = namespaceFile(resource)
    const requestTypes = requestTypesFor(
      context.program,
      ops,
      sdkOps,
      interfaces.refNameInput,
      bodyOverrides,
    )
    sdkFiles.push({
      path: `src/models/operations/${file}.ts`,
      content: operationsFile(resource, requestTypes),
    })
    const operationsAsserts = operationsAssertFile(resource, requestTypes)
    if (operationsAsserts) {
      sdkFiles.push({
        path: `src/models/operations/${file}.assert.ts`,
        content: operationsAsserts,
      })
    }
    sdkFiles.push({
      path: `src/funcs/${file}.ts`,
      content: funcsFile(resource, sdkOps),
    })
    sdkFiles.push({
      path: `src/sdk/${file}.ts`,
      content: facadeFile(resource, sdkOps),
    })
  }
  sdkFiles.push({ path: 'src/models/types.ts', content: interfaces.types })
  sdkFiles.push({
    path: 'src/models/types.assert.ts',
    content: interfaces.asserts,
  })
  sdkFiles.push({
    path: 'src/funcs/index.ts',
    content: funcsIndexFile(resources),
  })
  sdkFiles.push({ path: 'src/sdk/sdk.ts', content: sdkRootFile(resources) })
  sdkFiles.push({
    path: 'src/index.ts',
    content: indexFile(resources, interfaces.typeNames),
  })
  sdkFiles.push({
    path: 'README.md',
    content: readmeFile(
      readmeResources,
      context.options['package-name'],
      context.options['readme-note'],
    ),
  })

  writeOutput(
    context.program,
    <Output
      program={context.program}
      namePolicy={tsNamePolicy}
      externals={[zod]}
    >
      <ts.SourceFile path="src/models/schemas.ts">
        <ay.For
          each={types}
          ender={';'}
          joiner={
            <>
              ;<hbr />
              <hbr />
            </>
          }
        >
          {(type) => {
            const base = baseName(context.program, type)
            const name = base ? resolved.get(base) : undefined
            return <ZodSchemaDeclaration type={type} name={name} export />
          }}
        </ay.For>
        ;<hbr />
        <hbr />
        <ay.For
          each={opSchemas}
          ender={';'}
          joiner={
            <>
              ;<hbr />
              <hbr />
            </>
          }
        >
          {(schema) =>
            schema.render(resolved.get(schema.baseName) ?? schema.baseName)
          }
        </ay.For>
      </ts.SourceFile>
      {Object.entries(RUNTIME_TEMPLATES).map(([path, content]) => (
        <ts.SourceFile path={path}>{content}</ts.SourceFile>
      ))}
      {sdkFiles.map(({ path, content }) => (
        <ts.SourceFile path={path}>{content}</ts.SourceFile>
      ))}
    </Output>,
    context.emitterOutputDir,
  )
}

/**
 * Collects all the models defined in the spec and returns them in
 * topologically sorted order. Types are ordered such that dependencies appear
 * before the types that depend on them.
 */
function getAllDataTypes(program: Program) {
  const collector = newTopologicalTypeCollector(program)
  const globalNs = program.getGlobalNamespaceType()

  navigateProgram(
    program,
    {
      namespace(n) {
        if (n !== globalNs && !$(program).type.isUserDefined(n)) {
          return ListenerFlow.NoRecursion
        }
        return undefined
      },
      model: collector.collectType,
      enum: collector.collectType,
      union: collector.collectType,
      scalar: collector.collectType,
    },
    { includeTemplateDeclaration: false },
  )

  return collector.types
}
