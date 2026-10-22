import assert from 'node:assert/strict'
import { fileURLToPath } from 'node:url'
import { describe, it } from 'node:test'
import { createTester } from '@typespec/compiler/testing'

const tester = createTester(fileURLToPath(new URL('..', import.meta.url)), {
  libraries: ['@typespec/http', '@typespec/openapi3', '@typespec/versioning'],
})

describe('OpenAPI output paths', () => {
  for (const { version, filename } of [
    { version: '../../escape', filename: '.._.._escape.json' },
    { version: '..\\..\\escape', filename: '.._.._escape.json' },
    { version: '/tmp/escape', filename: '_tmp_escape.json' },
    { version: '2026-10-22', filename: '2026-10-22.json' },
  ]) {
    it(`keeps version ${JSON.stringify(version)} inside the output directory`, async () => {
      // Spec-derived names must not control directories, while benign versions
      // keep their existing filenames and OpenAPI metadata.
      const { outputs } = await tester.emit('@typespec/openapi3', {
        'output-file': '{version}.json',
      }).compile(`
          import "@typespec/http";
          import "@typespec/versioning";
          using TypeSpec.Http;
          using TypeSpec.Versioning;
          @service
          @versioned(Versions)
          namespace Service;
          enum Versions { v1: ${JSON.stringify(version)} }
          @get op read(): string;
        `)

      assert.deepEqual(Object.keys(outputs), [filename])
      assert.equal(JSON.parse(outputs[filename]).info.version, version)
    })
  }

  it('keeps spec-derived service names inside the output directory', async () => {
    const { outputs } = await tester.emit('@typespec/openapi3', {
      'output-file': '{service-name}.json',
    }).compile(`
        import "@typespec/http";
        using TypeSpec.Http;
        @service
        namespace \`../../escape\`;
        @get op read(): string;
      `)

    assert.deepEqual(Object.keys(outputs), ['.._.._escape.json'])
  })
})
