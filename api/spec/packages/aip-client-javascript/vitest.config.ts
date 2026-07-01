import { defineConfig } from 'vitest/config'

// The boundary mapper (src/lib/wire.ts) is the one hand-written runtime module with
// no compile-time guard, so its behavior is covered entirely by tests. Hold it at
// full statement/line/function coverage. The branch threshold is below 100 because
// the residual uncovered branches are defensive nullish-coalescing arms (`?? []`,
// `?? {}`) guarding zod internals that the preceding type-guard already makes
// non-null — unreachable with any real schema, so not worth contriving inputs for.
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      include: ['src/lib/wire.ts'],
      thresholds: {
        statements: 100,
        functions: 100,
        lines: 100,
        branches: 85,
      },
    },
  },
})
