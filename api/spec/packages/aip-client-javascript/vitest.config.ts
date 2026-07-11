import { defineConfig } from 'vitest/config'

// The boundary mapper (src/lib/wire.ts) and the pagination iterators
// (src/lib/paginate.ts) are the hand-written runtime modules with no
// compile-time guard, so their behavior is covered entirely by tests. Hold
// them at full statement/line/function coverage. The branch threshold is
// below 100 because wire.ts's residual uncovered branches are defensive
// nullish-coalescing arms (`?? []`, `?? {}`) guarding zod internals that the
// preceding type-guard already makes non-null — unreachable with any real
// schema, so not worth contriving inputs for. paginate.ts itself has no such
// residual: tests/paginate.spec.ts exercises every stop condition (empty
// page, short page, exact-total page, absent next cursor) and both
// PaginationLimitExceededError throws.
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      include: ['src/lib/wire.ts', 'src/lib/paginate.ts'],
      thresholds: {
        statements: 100,
        functions: 100,
        lines: 100,
        branches: 85,
      },
    },
  },
})
