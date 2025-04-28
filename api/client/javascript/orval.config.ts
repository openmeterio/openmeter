import { defineConfig } from 'orval'

export default defineConfig({
  openmeter: {
    input: {
      target: '../../openapi.cloud.yaml',
    },
    output: {
      clean: true,
      client: 'zod',
      mode: 'single',
      namingConvention: 'PascalCase',
      override: {
        header: () => `
/* eslint-disable no-useless-escape */
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
`,
        useDates: true,
        zod: {
          generate: {
            body: true,
            header: false,
            param: true,
            query: true,
            response: false,
          },
        },
      },
      prettier: true,
      propertySortOrder: 'Alphabetical',
      target: './src/zod/index.ts',
    },
  },
})
