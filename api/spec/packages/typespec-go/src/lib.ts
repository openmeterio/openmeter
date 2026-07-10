import { createTypeSpecLibrary, JSONSchemaType } from '@typespec/compiler'

export interface GoEmitterOptions {
  /** Go module path of the generated SDK, e.g. github.com/openmeterio/openmeter/sdk/go/openmeter. */
  'module-path': string
  /** Go package name for the flat single-package SDK, e.g. "openmeter". */
  'package-name': string
  /** Markdown inserted after the README intro, e.g. a GitHub alert callout. */
  'readme-note'?: string
  /**
   * Fallback SDK version used when Go build info is unavailable (the module
   * itself, replace directives, vendored trees without version data). Module
   * consumers instead get their resolved module version from
   * debug.ReadBuildInfo() at runtime, so tagged releases need no stamping
   * commit. Defaults to the 0.0.0-dev placeholder.
   */
  'sdk-version'?: string
  /**
   * Service namespace names whose operations are emitted as sub-clients. When
   * omitted, all services are included. Mirrors the TypeScript emitter's knob.
   */
  'include-services'?: string[]
  /** PascalCase type-name prefixes to strip when doing so is unambiguous. */
  'strip-name-prefixes'?: string[]
  /**
   * Operation-group names to generate. When omitted, every discovered group is
   * emitted.
   */
  'include-resources'?: string[]
  /**
   * Minimum Go version stamped into the generated go.mod's `go` directive.
   * Defaults to 1.23, the generated code's actual floor (the iter package).
   * Raise it when repo-preserved *_test.go files need newer stdlib APIs; doing
   * so raises the consumer floor for every SDK user, not just this repo.
   */
  'go-version'?: string
}

const EmitterOptionsSchema: JSONSchemaType<GoEmitterOptions> = {
  type: 'object',
  additionalProperties: true,
  properties: {
    'module-path': {
      type: 'string',
      description: 'Go module path of the generated SDK.',
    },
    'package-name': {
      type: 'string',
      description: 'Go package name for the flat single-package SDK.',
    },
    'readme-note': {
      type: 'string',
      nullable: true,
      description:
        'Markdown inserted after the README intro, e.g. a GitHub alert callout.',
    },
    'sdk-version': {
      type: 'string',
      nullable: true,
      description:
        'Fallback SDK version used when Go build info is unavailable (module consumers instead get their resolved module version at runtime). Defaults to 0.0.0-dev.',
    },
    'include-services': {
      type: 'array',
      items: { type: 'string' },
      nullable: true,
      description:
        'Service namespace names whose operations are emitted as sub-clients. When omitted, all services are included.',
    },
    'strip-name-prefixes': {
      type: 'array',
      items: { type: 'string' },
      nullable: true,
      description:
        'PascalCase type-name prefixes to strip when doing so is unambiguous.',
    },
    'include-resources': {
      type: 'array',
      items: { type: 'string' },
      nullable: true,
      description:
        'Operation-group names to generate. Defaults to every discovered group.',
    },
    'go-version': {
      type: 'string',
      nullable: true,
      description:
        "Minimum Go version stamped into the generated go.mod's go directive. Defaults to 1.23, the generated code's actual floor (the iter package). Raise it when repo-preserved *_test.go files need newer stdlib APIs, noting it raises the consumer floor.",
    },
  },
  required: ['module-path', 'package-name'],
}

export const $lib = createTypeSpecLibrary({
  name: 'typespec-go',
  emitter: {
    options: EmitterOptionsSchema,
  },
  diagnostics: {},
})

export const { reportDiagnostic, createDiagnostic } = $lib
