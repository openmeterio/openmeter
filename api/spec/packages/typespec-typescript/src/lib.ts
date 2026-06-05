import { createTypeSpecLibrary, JSONSchemaType } from '@typespec/compiler'

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
  diagnostics: {},
})

export const { reportDiagnostic, createDiagnostic } = $lib
