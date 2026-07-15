import { fileURLToPath } from 'node:url'
import { createTester } from '@typespec/compiler/testing'

// The tester resolves the emitter through its package.json entry point, so
// tests exercise the BUILT emitter (dist/) end to end through the compiler's
// real emit pipeline — the `test` script builds first to keep dist fresh.
const packageRoot = fileURLToPath(new URL('..', import.meta.url))

const Tester = createTester(packageRoot, {
  libraries: [
    '@typespec/http',
    '@typespec/openapi',
    '@openmeter/typespec-sdk',
    '@openmeter/typespec-typescript',
  ],
})

/** Compiles a fixture spec and returns the emitted SDK files keyed by path. */
export const EmitterTester = Tester.emit('@openmeter/typespec-typescript', {
  'package-name': '@test/client',
})
