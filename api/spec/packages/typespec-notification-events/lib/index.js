import {
  createTypeSpecLibrary,
  emitFile,
  getFriendlyName,
  getNamespaceFullName,
  paramMessage,
  resolvePath,
} from '@typespec/compiler'
import { getId, isJsonSchemaDeclaration } from '@typespec/json-schema'
import { posix } from 'node:path'

export const $lib = createTypeSpecLibrary({
  name: '@openmeter/typespec-notification-events',
  diagnostics: {
    'invalid-event-type': {
      severity: 'error',
      messages: {
        default: paramMessage`Event type "${'eventType'}" must be v<version>.<name> with a positive version and at least two lowercase snake_case name segments.`,
      },
    },
    'invalid-contract': {
      severity: 'error',
      messages: {
        default: paramMessage`${'detail'}`,
      },
    },
    'event-type-outside-go-package': {
      severity: 'error',
      messages: {
        default:
          'Event payloads must be declared directly in a namespace decorated with @goPackage.',
      },
    },
  },
  emitter: {
    options: {
      type: 'object',
      additionalProperties: false,
      properties: {
        'module-path': { type: 'string' },
        'eventtype-import-path': { type: 'string' },
        'definitions-import-path': { type: 'string' },
        'oapi-codegen-config': { type: 'string' },
        'spec-dir': { type: 'string' },
      },
      required: [
        'module-path',
        'eventtype-import-path',
        'definitions-import-path',
        'oapi-codegen-config',
        'spec-dir',
      ],
    },
  },
})

const goPackageKey = $lib.createStateSymbol('goPackage')
const eventTypeKey = $lib.createStateSymbol('eventType')

// Canonical Event type: v<version>.<name>, where the name has at least two
// lowercase snake_case segments (<resource>.<operation>).
const eventTypePattern =
  /^v([1-9][0-9]*)\.([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+)$/

export const $decorators = {
  NotificationCodegen: {
    goPackage(context, target, path) {
      context.program.stateMap(goPackageKey).set(target, { path })
    },
    eventType(context, target, eventType, ruleConfig) {
      const match = eventTypePattern.exec(eventType)
      if (!match) {
        context.program.reportDiagnostic(
          $lib.createDiagnostic({
            code: 'invalid-event-type',
            format: { eventType },
            target: context.decoratorTarget,
          }),
        )
        return
      }
      context.program.stateMap(eventTypeKey).set(target, {
        eventType,
        version: Number(match[1]),
        name: match[2],
        ruleConfig,
      })
    },
  },
}

// The Go identifier oapi-codegen derives from the OpenAPI component name,
// which the AIP linter requires to be set through @friendlyName.
const goName = (program, type) => getFriendlyName(program, type) ?? type.name

// Released contracts are immutable and identified by their canonical Event
// type, so the contract roots carry it in every name: the JSON Schema $id
// (<event type>.payload / <event type>.rule_config) and the friendly name
// (...V<version>Payload / ...V<version>RuleConfig, sharing one stem).
// Nested types stay unversioned; changing one requires a new type.
export function $onValidate(program) {
  const packages = program.stateMap(goPackageKey)
  const invalid = (target, detail) =>
    program.reportDiagnostic(
      $lib.createDiagnostic({
        code: 'invalid-contract',
        format: { detail },
        target,
      }),
    )

  for (const [model, e] of program.stateMap(eventTypeKey)) {
    if (!packages.has(model.namespace)) {
      program.reportDiagnostic(
        $lib.createDiagnostic({
          code: 'event-type-outside-go-package',
          target: model,
        }),
      )
    }

    const payload = goName(program, model)
    const payloadSuffix = `V${e.version}Payload`
    const hasPayloadSuffix = payload.endsWith(payloadSuffix)
    if (!hasPayloadSuffix) {
      invalid(
        model,
        `Payload ${payload} of ${e.eventType} must have a friendly name ending in ${payloadSuffix}.`,
      )
    }

    const roots = [[model, `${e.eventType}.payload`]]
    if (e.ruleConfig) {
      roots.push([e.ruleConfig, `${e.eventType}.rule_config`])
      // The expected name derives from the payload's, so it is only
      // meaningful once the payload name itself is valid.
      const expected = payload.replace(/Payload$/, 'RuleConfig')
      if (hasPayloadSuffix && goName(program, e.ruleConfig) !== expected) {
        invalid(
          e.ruleConfig,
          `Rule configuration of ${e.eventType} must have the friendly name ${expected}.`,
        )
      }
    }
    for (const [type, id] of roots) {
      if (!isJsonSchemaDeclaration(program, type)) {
        invalid(type, `${type.name} must be decorated with @jsonSchema.`)
      }
      if (getId(program, type) !== id) {
        invalid(type, `${type.name} must be decorated with @id("${id}").`)
      }
    }
  }
}

// Emits, under the emitter output directory:
//   - <go package dir>/generate.gen.go for every @goPackage namespace, holding
//     the go:generate directive that runs oapi-codegen on the package's
//     events-only OpenAPI document with the shared configuration;
//   - <go package dir>/events.gen.go for every package owning @eventType
//     payloads: the Event type value, the embedded JSON Schemas and the
//     Event type definition;
//   - <definitions package dir>/definitions.gen.go listing the definitions of
//     every package, so the Event Registry and contract checks cannot miss a
//     type;
//   - manifest.json describing every @goPackage namespace and the definitions
//     package, used by scripts/generate-notification-events.mjs to generate
//     the events-only OpenAPI documents and copy them, the Go files and the
//     schemas into the repository.
//
// Paths in options and in the manifest are relative to the repository root.
export async function $onEmit(context) {
  const program = context.program
  const modulePath = context.options['module-path']
  const eventTypeImportPath = context.options['eventtype-import-path']
  const eventTypePkg = eventTypeImportPath.split('/').pop()
  const definitionsImportPath = context.options['definitions-import-path']
  const oapiCodegenConfig = context.options['oapi-codegen-config']
  const specDir = context.options['spec-dir']

  const eventsByNamespace = new Map()
  for (const [model, def] of program.stateMap(eventTypeKey)) {
    const list = eventsByNamespace.get(model.namespace) ?? []
    list.push({ model, ...def })
    eventsByNamespace.set(model.namespace, list)
  }

  const moduleDir = (path) => {
    if (!path.startsWith(modulePath + '/')) {
      throw new Error(`Go package ${path} is outside module ${modulePath}`)
    }
    return path.slice(modulePath.length + 1)
  }

  const packages = []
  // Every generated definition, as [Go import alias, import path, identifier, event type].
  const definitions = []
  for (const [ns, { path }] of program.stateMap(goPackageKey)) {
    const dir = moduleDir(path)
    const pkgName = path.split('/').pop()
    // openmeter/entitlement/notificationevents -> <spec-dir>/entitlement/openapi.yaml
    const domain = dir.replace(/^openmeter\//, '').replace(/\/[^/]+$/, '')
    // openmeter/entitlement/notificationevents -> entitlementevents
    const importAlias = `${domain.replaceAll('/', '')}events`
    const spec = posix.join(specDir, domain, 'openapi.yaml')
    const header = [
      '// Code generated by @openmeter/typespec-notification-events, DO NOT EDIT.',
      '',
      `package ${pkgName}`,
      '',
    ]

    await emitFile(program, {
      path: resolvePath(context.emitterOutputDir, dir, 'generate.gen.go'),
      content: [
        ...header,
        [
          '//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen',
          `-config ${posix.relative(dir, oapiCodegenConfig)}`,
          `-package ${pkgName}`,
          '-o payloads.gen.go',
          posix.relative(dir, spec),
        ].join(' '),
        '',
      ].join('\n'),
    })

    const schemas = []
    const events = (eventsByNamespace.get(ns) ?? []).sort((a, b) =>
      a.eventType.localeCompare(b.eventType),
    )
    if (events.length > 0) {
      const lines = [
        ...header,
        'import (',
        '\t_ "embed"',
        '',
        `\t"${eventTypeImportPath}"`,
        ')',
        '',
      ]
      for (const e of events) {
        const payload = goName(program, e.model)
        const stem = payload.replace(/Payload$/, '')
        // @typespec/json-schema names files after the TypeSpec declaration;
        // the copy into the Go package renames them after their $id.
        const payloadSchema = `${e.eventType}.payload.json`
        schemas.push({ source: `${e.model.name}.json`, target: payloadSchema })
        lines.push(
          `// ${stem}EventType is ${e.eventType}.`,
          `var ${stem}EventType = ${eventTypePkg}.EventType{`,
          `\tName:    "${e.name}",`,
          `\tVersion: ${e.version},`,
          '}',
          '',
          `// ${stem}PayloadJSONSchema is the JSON Schema (draft 2020-12) of ${payload}.`,
          '//',
          `//go:embed schemas/${payloadSchema}`,
          `var ${stem}PayloadJSONSchema []byte`,
          '',
        )
        if (e.ruleConfig) {
          const ruleConfigSchema = `${e.eventType}.rule_config.json`
          schemas.push({
            source: `${e.ruleConfig.name}.json`,
            target: ruleConfigSchema,
          })
          lines.push(
            `// ${stem}RuleConfigJSONSchema is the JSON Schema (draft 2020-12) of`,
            `// ${goName(program, e.ruleConfig)}, the Rule configuration of ${e.eventType}.`,
            '//',
            `//go:embed schemas/${ruleConfigSchema}`,
            `var ${stem}RuleConfigJSONSchema []byte`,
            '',
          )
        }
        lines.push(
          `// ${stem}Definition is the definition of ${e.eventType}.`,
          `var ${stem}Definition = ${eventTypePkg}.EventTypeDefinition{`,
          `\tEventType:         ${stem}EventType,`,
          `\tPayloadJSONSchema: ${stem}PayloadJSONSchema,`,
          ...(e.ruleConfig
            ? [
                `\tRule: &${eventTypePkg}.RuleConfigDefinition{`,
                `\t\tJSONSchema: ${stem}RuleConfigJSONSchema,`,
                '\t},',
              ]
            : []),
          '}',
          '',
        )
        definitions.push([importAlias, path, `${stem}Definition`, e.eventType])
      }
      await emitFile(program, {
        path: resolvePath(context.emitterOutputDir, dir, 'events.gen.go'),
        content: lines.join('\n'),
      })
    }

    packages.push({
      namespace: getNamespaceFullName(ns),
      dir,
      package: pkgName,
      spec,
      schemas,
    })
  }

  const definitionsDir = moduleDir(definitionsImportPath)
  const definitionsPkg = definitionsImportPath.split('/').pop()
  definitions.sort((a, b) => a[3].localeCompare(b[3]))
  const imports = [
    ...new Map(definitions.map(([alias, path]) => [path, alias])),
    [eventTypeImportPath, ''],
  ].sort(([a], [b]) => a.localeCompare(b))
  await emitFile(program, {
    path: resolvePath(
      context.emitterOutputDir,
      definitionsDir,
      'definitions.gen.go',
    ),
    content: [
      '// Code generated by @openmeter/typespec-notification-events, DO NOT EDIT.',
      '',
      `package ${definitionsPkg}`,
      '',
      'import (',
      ...imports.map(([path, alias]) =>
        alias ? `\t${alias} "${path}"` : `\t"${path}"`,
      ),
      ')',
      '',
      '// Definitions returns the definition of every Notification Event type.',
      `func Definitions() []${eventTypePkg}.EventTypeDefinition {`,
      `\treturn []${eventTypePkg}.EventTypeDefinition{`,
      ...definitions.map(
        ([alias, , identifier]) => `\t\t${alias}.${identifier},`,
      ),
      '\t}',
      '}',
      '',
    ].join('\n'),
  })

  await emitFile(program, {
    path: resolvePath(context.emitterOutputDir, 'manifest.json'),
    content:
      JSON.stringify({ packages, definitions: definitionsDir }, null, 2) + '\n',
  })
}
