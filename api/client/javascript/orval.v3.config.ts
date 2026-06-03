import { defineConfig } from 'orval'

// v3 compatibility shim — Zod schema generator. Mirrors orval.config.ts but
// reads the v3 OpenAPI spec (api/v3/openapi.yaml) and writes src/v3/zod/index.ts.
// See V3_SHIM_PLAN.md.
export default defineConfig({
  openmeter: {
    input: {
      target: '../../v3/openapi.yaml',
    },
    output: {
      formatter: 'biome',
      clean: true,
      client: 'zod',
      mode: 'single',
      namingConvention: 'PascalCase',
      override: {
        useDates: true,
        zod: {
          coerce: {
            body: true,
            header: false,
            param: true,
            query: true,
            response: false,
          },
          generate: {
            body: true,
            header: false,
            param: true,
            query: true,
            response: false,
          },
        },
      },
      propertySortOrder: 'Alphabetical',
      target: './src/v3/zod/index.ts',
      tsconfig: './tsconfig.json',
    },
  },
})
