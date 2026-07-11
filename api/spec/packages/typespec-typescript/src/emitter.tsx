import * as ay from '@alloy-js/core'
import * as ts from '@alloy-js/typescript'
import {
  type EmitContext,
  getFriendlyName,
  ListenerFlow,
  type Model,
  navigateProgram,
  type Operation,
  type Program,
  type Type,
  type Union,
} from '@typespec/compiler'
import { $ } from '@typespec/compiler/typekit'
// Registers the experimental HTTP typekit ($.httpOperation, $.modelProperty
// HTTP helpers) used by the operation walk and metadata stripping.
import '@typespec/http/experimental/typekit'
import { Output, writeOutput } from '@typespec/emitter-framework'
import { ZodSchemaDeclaration } from './components/ZodSchemaDeclaration.jsx'
import { zod } from './external-packages/zod.js'
import type { ZodEmitterOptions } from './lib.js'
import {
  collectHttpOperations,
  isInternalOperation,
  operationSchemas,
} from './ZodOperations.jsx'
import { resolveStrippedNames } from './strip-prefixes.js'
import { newTopologicalTypeCollector, WireModeContext } from './utils.jsx'
import { RUNTIME_TEMPLATES } from './runtime-templates.js'
import { assertCasingDerivable } from './casing-gate.js'
import { WIRE_RUNTIME } from './wire-runtime.js'
import { interfacesFile } from './interface-types.js'
import { inputVariantName } from './input-variants.js'
import {
  computeResponseReachableModels,
  setResponseReachableModels,
} from './visibility.js'
import { findPaginationTemplates, paginationInfo } from './pagination.js'
import {
  groupOperations,
  jsonBodyOverrides,
  sdkOperation,
  type SdkOperation,
} from './sdk-operations.js'
import { requestTypesFor } from './request-types.js'
import { readmeFile, type ReadmeResource } from './readme.js'
import {
  facadeFile,
  funcsFile,
  funcsIndexFile,
  indexFile,
  internalFile,
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

  // Gate visibility filtering on response reachability before any model is
  // walked: a create-/update-only property is dropped from a model's read shape
  // only when that model appears in a response body. Set once, consulted by
  // every property walker (interfaces, zod schemas, input-variant detection).
  setResponseReachableModels(
    context.program,
    computeResponseReachableModels(context.program, operations),
  )

  const bodyOverrides = jsonBodyOverrides(context.program)
  const opSchemas = operations.flatMap((op) =>
    operationSchemas(context.program, op, bodyOverrides),
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
  // A union can be declared in TypeSpec — and still picked up as a zod schema,
  // since `getAllDataTypes` walks the whole namespace tree — without anything
  // in the SDK surface ever referencing it (`PriceUsageBased` is reserved for
  // future use; `ULIDOrResourceKey`/`ULIDOrExternalResourceKey` are unused
  // today). Such a union has no meaningful shape for an SDK user to import
  // (often literally `string | string`), so it is excluded from the `types.ts`
  // alias pass even though its zod schema is still emitted.
  const reachableUnions = computeReachableUnions(context.program, operations)
  const unions = types.filter(
    (t): t is Union => t.kind === 'Union' && reachableUnions.has(t),
  )

  // Fail the build if any wire key is not recoverable from its camelCase public
  // form by the deterministic casing rule, before emitting anything that relies
  // on it.
  assertCasingDerivable(context.program, models, operations)

  const interfaceName = (name: string) =>
    tsNamePolicy.getName(name, 'interface')
  const interfaces = interfacesFile(
    context.program,
    models,
    unions,
    resolveName,
    (name) => tsNamePolicy.getName(name, 'variable'),
    interfaceName,
  )

  // A type resolves to its documented interface only when its resolved name
  // matches an emitted interface (string-based, so template instantiations whose
  // Type identity differs from the collected model still resolve).
  const emittedInterfaceNames = new Set(
    [...models, ...unions]
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
    [...interfaces.divergentModels, ...interfaces.divergentUnions]
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

  const groups = groupOperations(context.program, operations)
  const resources = [...groups.keys()]
  const sdkFiles: Array<{ path: string; content: string }> = []
  const readmeResources: ReadmeResource[] = []
  // Groups' x-internal operations, destined for the `client.internal.*`
  // surface. They share their group's funcs and operations modules with the
  // public ops; only the facade classes are split. A group whose operations
  // are all internal (e.g. currencies) gets no public facade at all.
  const internalResources: ReadmeResource[] = []
  const publicResources: string[] = []
  const paginationTemplates = findPaginationTemplates(context.program)
  // Names the operations modules export (`<Base>Request`/`<Base>Response`/
  // `<Base>Query`), mirroring requestDecl/queryType/responseDecl in
  // request-types.ts. indexFile must not re-export a same-named domain model:
  // the explicit re-export would shadow the operation alias at the package
  // root, and the two are not interchangeable (the alias wraps path params and
  // AcceptDateStrings widening).
  const operationTypeNames = new Set<string>()
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
    ops.forEach((op, i) => {
      sdkOps[i]!.pagination = paginationInfo(
        context.program,
        op,
        paginationTemplates,
        resolveInterface,
      )
    })
    for (const op of sdkOps) {
      operationTypeNames.add(`${op.base}Request`)
      operationTypeNames.add(`${op.base}Response`)
      if (op.queryParams.length > 0) {
        operationTypeNames.add(`${op.base}Query`)
      }
    }
    const publicOps: SdkOperation[] = []
    const internalOps: SdkOperation[] = []
    ops.forEach((op, i) => {
      const target = isInternalOperation(context.program, op)
        ? internalOps
        : publicOps
      target.push(sdkOps[i]!)
    })
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
    if (publicOps.length > 0) {
      publicResources.push(resource)
      readmeResources.push({ resource, ops: publicOps })
      sdkFiles.push({
        path: `src/sdk/${file}.ts`,
        content: facadeFile(resource, publicOps),
      })
    }
    if (internalOps.length > 0) {
      internalResources.push({ resource, ops: internalOps })
    }
  }
  if (internalResources.length > 0) {
    sdkFiles.push({
      path: 'src/sdk/internal.ts',
      content: internalFile(internalResources),
    })
  }
  sdkFiles.push({ path: 'src/lib/wire.ts', content: WIRE_RUNTIME })
  sdkFiles.push({ path: 'src/models/types.ts', content: interfaces.types })
  sdkFiles.push({
    path: 'src/models/types.assert.ts',
    content: interfaces.asserts,
  })
  sdkFiles.push({
    path: 'src/funcs/index.ts',
    content: funcsIndexFile(resources),
  })
  sdkFiles.push({
    path: 'src/sdk/sdk.ts',
    content: sdkRootFile(publicResources, internalResources.length > 0),
  })
  sdkFiles.push({
    path: 'src/index.ts',
    content: indexFile(
      publicResources,
      resources,
      interfaces.typeNames,
      operationTypeNames,
    ),
  })
  sdkFiles.push({
    path: 'README.md',
    content: readmeFile(
      readmeResources,
      context.options['package-name'],
      context.options['readme-note'],
      internalResources,
    ),
  })

  // Stamped on every emitted file (repo convention for generated code): the
  // four regenerated tests/ files and README.md are indistinguishable from
  // hand-written files without it, inviting edits the next generate reverts.
  const generatedComment =
    'Code generated by @openmeter/typespec-typescript. DO NOT EDIT.'

  writeOutput(
    context.program,
    <Output
      program={context.program}
      namePolicy={tsNamePolicy}
      externals={[zod]}
    >
      <ts.SourceFile
        path="src/models/schemas.ts"
        headerComment={generatedComment}
      >
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
        ;<hbr />
        <hbr />
        {/* The snake_case wire pass: the same models and per-op body/response
            schemas re-emitted strict for the optional `validate` option. Because
            both come from one walk over the same types, they are structurally
            identical except for casing and strictness. */}
        <WireModeContext.Provider value={true}>
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
              return (
                <ZodSchemaDeclaration
                  type={type}
                  name={name ? `${name}Wire` : undefined}
                  export
                />
              )
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
              schema.render(
                `${resolved.get(schema.baseName) ?? schema.baseName}Wire`,
              )
            }
          </ay.For>
        </WireModeContext.Provider>
      </ts.SourceFile>
      {Object.entries(RUNTIME_TEMPLATES).map(([path, content]) => (
        <ts.SourceFile path={path} headerComment={generatedComment}>
          {content}
        </ts.SourceFile>
      ))}
      {sdkFiles.map(({ path, content }) =>
        path.endsWith('.md') ? (
          <ts.SourceFile path={path}>
            {`<!-- ${generatedComment} -->\n\n${content}`}
          </ts.SourceFile>
        ) : (
          <ts.SourceFile path={path} headerComment={generatedComment}>
            {content}
          </ts.SourceFile>
        ),
      )}
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

/**
 * Named unions transitively reachable from any operation's request body, query
 * parameters, or response body (success or error) — descending through model
 * properties, indexers, base/derived models, and nested union variants, the
 * same shape the interface/schema walkers traverse. Used to gate which unions
 * earn a `types.ts` alias: a declared-but-unreferenced union is still
 * collected as a zod schema (see `getAllDataTypes`), but has nothing to alias
 * to that an SDK caller could actually encounter.
 */
function computeReachableUnions(
  program: Program,
  operations: Operation[],
): Set<Union> {
  const reachable = new Set<Union>()
  const visited = new Set<Type>()

  const visit = (type: Type | undefined): void => {
    if (!type || visited.has(type)) {
      return
    }
    visited.add(type)
    switch (type.kind) {
      case 'Model':
        if (type.indexer) {
          visit(type.indexer.value)
        }
        if (type.baseModel) {
          visit(type.baseModel)
        }
        for (const derived of type.derivedModels) {
          visit(derived)
        }
        for (const prop of type.properties.values()) {
          visit(prop.type)
        }
        break
      case 'Union':
        reachable.add(type)
        for (const variant of type.variants.values()) {
          visit(variant.type)
        }
        break
      case 'Tuple':
        for (const value of type.values) {
          visit(value)
        }
        break
      default:
        break
    }
  }

  const tk = $(program)
  for (const op of operations) {
    const httpOp = tk.httpOperation.get(op)
    visit(httpOp.parameters.body?.type)
    for (const param of httpOp.parameters.parameters) {
      if (param.type === 'query') {
        visit(param.param.type)
      }
    }
    for (const response of httpOp.responses) {
      for (const content of response.responses) {
        visit(content.body?.type)
      }
    }
  }

  return reachable
}
