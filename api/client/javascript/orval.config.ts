import { defineConfig } from 'orval'

export default defineConfig({
  openmeter: {
    input: {
      target: '../../openapi.cloud.yaml',
    },
    output: {
      biome: true,
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
      target: './src/zod/index.ts',
      tsconfig: './tsconfig.json',
    },
  },
})
