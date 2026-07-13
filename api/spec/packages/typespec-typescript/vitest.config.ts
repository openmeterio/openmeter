import { configDefaults, defineConfig } from 'vitest/config'

// `templates/` holds real, committed copies of the SDK runtime files and
// conformance tests that `runtime-templates.ts` reads via readFileSync and
// emits into the generated SDK (see api/spec/AGENTS.md). Vitest's default
// include glob (`**/*.spec.ts`) would otherwise collect `templates/tests/*.spec.ts`
// as this package's own tests — they import `../src/index.js`, which only
// exists in the generated `aip-client-javascript` output where these files
// are actually emitted and run (via `pnpm run test:sdk`).
export default defineConfig({
  test: {
    exclude: [...configDefaults.exclude, 'templates/**'],
  },
})
