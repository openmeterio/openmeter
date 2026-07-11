// Static SDK runtime files and conformance tests, copied verbatim into the
// generated SDK. The source of truth is the committed files under
// `templates/` — real, reviewable `.ts` files, not embedded blobs. They are
// excluded from this package's own typecheck (see tsconfig.json's `include`)
// because they are checked where they actually run: as part of the generated
// `aip-client-javascript` package's own typecheck/test suite.
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

// outputPath (in the generated SDK) -> template file under `templates/`.
// Runtime files are fixed boilerplate; the conformance tests are the spec the
// generated SDK must satisfy and are emitted alongside it so `test:sdk` runs
// against generated output.
const RUNTIME_FILES: Record<string, string> = {
  'src/core.ts': 'runtime/core.ts',
  'src/lib/types.ts': 'runtime/types.ts',
  'src/lib/config.ts': 'runtime/config.ts',
  'src/lib/version.ts': 'runtime/version.ts',
  'src/lib/encodings.ts': 'runtime/encodings.ts',
  'src/lib/to-error.ts': 'runtime/to-error.ts',
  'src/lib/request.ts': 'runtime/request.ts',
  'src/lib/paginate.ts': 'runtime/paginate.ts',
  'src/models/errors.ts': 'runtime/errors.ts',
  'tests/client.spec.ts': 'tests/client.spec.ts',
  'tests/meters.spec.ts': 'tests/meters.spec.ts',
  'tests/errors.spec.ts': 'tests/errors.spec.ts',
  'tests/nesting.spec.ts': 'tests/nesting.spec.ts',
}

export const RUNTIME_TEMPLATES: Record<string, string> = Object.fromEntries(
  Object.entries(RUNTIME_FILES).map(([outPath, templateRelPath]) => [
    outPath,
    readFileSync(
      fileURLToPath(
        new URL(`../templates/${templateRelPath}`, import.meta.url),
      ),
      'utf8',
    ),
  ]),
)
