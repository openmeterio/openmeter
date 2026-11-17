import {
  createTypeSpecLibrary,
  emitFile,
  getDoc,
  getFriendlyName,
  paramMessage,
  resolvePath,
} from '@typespec/compiler'
import { getId, isJsonSchemaDeclaration } from '@typespec/json-schema'

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
  },
  emitter: {
    options: {
      type: 'object',
      additionalProperties: false,
      properties: {
        'module-path': { type: 'string' },
        'eventtype-import-path': { type: 'string' },
        'definitions-import-path': { type: 'string' },
      },
      required: [
        'module-path',
        'eventtype-import-path',
        'definitions-import-path',
      ],
    },
  },
})

const eventTypeKey = $lib.createStateSymbol('eventType')

// Canonical Event type: v<version>.<name>, where the name has at least two
// lowercase snake_case segments (<resource>.<operation>).
const eventTypePattern =
  /^v([1-9][0-9]*)\.([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+)$/

export const $decorators = {
  NotificationCodegen: {
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

// Compares by UTF-16 code units, unlike localeCompare, so generated files are
// ordered the same on every machine.
const compareStrings = (a, b) => (a < b ? -1 : a > b ? 1 : 0)

// Orders parsed Event types like v4.EventType.Compare: numeric version, then
// name, so v2 comes before v10.
const compareEventTypes = (a, b) =>
  a.version - b.version || compareStrings(a.name, b.name)

// Released contracts are immutable and identified by their canonical Event
// type, so the contract roots carry it in every name: the JSON Schema $id
// (<event type>.payload / <event type>.rule_config) and the friendly name
// (...V<version>Payload / ...V<version>RuleConfig, sharing one stem).
// Nested types stay unversioned; changing one requires a new type.
export function $onValidate(program) {
  const invalid = (target, detail) =>
    program.reportDiagnostic(
      $lib.createDiagnostic({
        code: 'invalid-contract',
        format: { detail },
        target,
      }),
    )

  // Every definition lands in one Go package and its identifiers derive from
  // the payload name, so two contracts sharing it would not compile. The Rule
  // configuration name derives from the payload name too.
  const eventTypeByPayload = new Map()

  for (const [model, e] of program.stateMap(eventTypeKey)) {
    const payload = goName(program, model)
    const payloadSuffix = `V${e.version}Payload`
    const hasPayloadSuffix = payload.endsWith(payloadSuffix)
    if (!hasPayloadSuffix) {
      invalid(
        model,
        `Payload ${payload} of ${e.eventType} must have a friendly name ending in ${payloadSuffix}.`,
      )
    }

    const other = eventTypeByPayload.get(payload)
    if (other) {
      invalid(
        model,
        `Payload ${payload} of ${e.eventType} has the same friendly name as the payload of ${other}.`,
      )
    } else {
      eventTypeByPayload.set(payload, e.eventType)
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
//   - <definitions package dir>/definitions.gen.go holding, for every Event
//     type, its EventType value, embedded JSON Schemas and definition, and
//     All() listing every definition, so the Event Registry and contract
//     checks cannot miss a type;
//   - manifest.json naming the definitions package and the JSON Schemas to
//     copy into its schemas directory, used by
//     scripts/generate-notification-events.mjs.
//
// The payload and Rule configuration Go types are not generated here: the v3
// API references them, so they are generated into api/v3/models.
//
// Paths in options and in the manifest are relative to the repository root.
export async function $onEmit(context) {
  const program = context.program
  const modulePath = context.options['module-path']
  const eventTypeImportPath = context.options['eventtype-import-path']
  const eventTypePkg = eventTypeImportPath.split('/').pop()
  const definitionsImportPath = context.options['definitions-import-path']

  if (!definitionsImportPath.startsWith(modulePath + '/')) {
    throw new Error(
      `Go package ${definitionsImportPath} is outside module ${modulePath}`,
    )
  }
  const definitionsDir = definitionsImportPath.slice(modulePath.length + 1)
  const definitionsPkg = definitionsImportPath.split('/').pop()

  const events = [...program.stateMap(eventTypeKey)]
    .map(([model, def]) => ({ model, ...def }))
    .sort(compareEventTypes)

  const schemas = []
  const lines = [
    '// Code generated by @openmeter/typespec-notification-events, DO NOT EDIT.',
    '',
    `package ${definitionsPkg}`,
    '',
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
    // @typespec/json-schema names files after the TypeSpec declaration; the
    // copy into the Go package renames them after their $id.
    const payloadSchema = `${e.eventType}.payload.json`
    schemas.push({ source: `${e.model.name}.json`, target: payloadSchema })
    lines.push(
      `// ${stem}EventType is ${e.eventType}.`,
      `var ${stem}EventType = ${eventTypePkg}.EventType{`,
      `\tName:    "${e.name}",`,
      `\tVersion: ${e.version},`,
      '}',
      '',
      `// ${stem}PayloadJSONSchema is the JSON Schema (draft 2020-12) of`,
      `// apiv3models.${payload}.`,
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
      // The payload's doc comment describes the Event type. JSON string
      // escaping is valid Go string literal syntax.
      `\tDescription:       ${JSON.stringify(getDoc(program, e.model) ?? '')},`,
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
  }
  lines.push(
    '// All returns the definition of every Notification Event type.',
    `func All() []${eventTypePkg}.EventTypeDefinition {`,
    // gofmt writes an empty composite literal on one line.
    ...(events.length === 0
      ? [`\treturn []${eventTypePkg}.EventTypeDefinition{}`]
      : [
          `\treturn []${eventTypePkg}.EventTypeDefinition{`,
          ...events.map(
            (e) =>
              `\t\t${goName(program, e.model).replace(/Payload$/, '')}Definition,`,
          ),
          '\t}',
        ]),
    '}',
    '',
  )

  await emitFile(program, {
    path: resolvePath(
      context.emitterOutputDir,
      definitionsDir,
      'definitions.gen.go',
    ),
    content: lines.join('\n'),
  })

  await emitFile(program, {
    path: resolvePath(context.emitterOutputDir, 'manifest.json'),
    content:
      JSON.stringify({ definitions: definitionsDir, schemas }, null, 2) + '\n',
  })
}
