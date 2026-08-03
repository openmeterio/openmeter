import {
  createTypeSpecLibrary,
  JSONSchemaType,
  paramMessage,
} from '@typespec/compiler'

export interface ZodEmitterOptions {
  'package-name': string
  'readme-note'?: string
  'strip-name-prefixes'?: string[]
  'include-services'?: string[]
}

const EmitterOptionsSchema: JSONSchemaType<ZodEmitterOptions> = {
  type: 'object',
  additionalProperties: true,
  properties: {
    'package-name': {
      type: 'string',
      description:
        'The npm package name the generated README installs and imports.',
    },
    'readme-note': {
      type: 'string',
      nullable: true,
      description:
        'Markdown inserted after the README intro, e.g. a GitHub alert callout.',
    },
    'strip-name-prefixes': {
      type: 'array',
      items: { type: 'string' },
      nullable: true,
      description: 'Prefixes to strip from generated Zod schema',
    },
    'include-services': {
      type: 'array',
      items: { type: 'string' },
      nullable: true,
      description:
        'Service namespace names whose operations get per-operation schemas. When omitted, all services are included.',
    },
  },
  required: ['package-name'],
}

export const $lib = createTypeSpecLibrary({
  name: 'typespec-typescript',
  emitter: {
    options: EmitterOptionsSchema,
  },
  // Every silent-degradation path in the emitter must report one of these.
  // Nothing in the current spec triggers them — they exist so future spec
  // shapes the emitter cannot faithfully map surface in the generate output
  // instead of quietly eroding the SDK (a z.any() schema, or an endpoint
  // missing from the published client).
  diagnostics: {
    'unsupported-type': {
      severity: 'warning',
      messages: {
        default: paramMessage`${'kind'} has no schema mapping and degrades to any/unknown in the generated SDK`,
      },
    },
    'ungrouped-operation': {
      severity: 'warning',
      messages: {
        default: paramMessage`operation '${'operation'}' has no resolvable source interface (not authored via the extends/op-is pattern) and was omitted from the generated SDK`,
      },
    },
  },
})

export const { reportDiagnostic, createDiagnostic } = $lib
