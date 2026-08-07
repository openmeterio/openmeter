import type { Model, Program } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { describe, expect, it } from 'vitest'
import { conflictingVariantProperty } from '../dist/components/GoUnion.js'

async function compileModels(code: string): Promise<{
  program: Program
  variants: (...names: string[]) => Model[]
}> {
  const host = await createTestHost({
    libraries: [HttpTestLibrary, OpenAPITestLibrary],
  })
  const runner = await createTestRunner(host)
  await runner.compile(`
    using TypeSpec.Http;
    using TypeSpec.OpenAPI;
    ${code}
  `)
  const models = runner.program
    .getGlobalNamespaceType()
    .namespaces.get('Test')!.models
  return {
    program: runner.program,
    variants: (...names) => names.map((name) => models.get(name)!),
  }
}

describe('non-discriminated union variant compatibility', () => {
  it('accepts variants whose properties are a subset of another variant', async () => {
    const { program, variants } = await compileModels(`
      @service namespace Test;

      model OwnerRef {
        id: string;
      }

      model Owner {
        id: string;
        display_name: string;
      }
    `)

    expect(
      conflictingVariantProperty(program, variants('Owner', 'OwnerRef')),
    ).toBeUndefined()
  })

  it('reports a JSON key two variants declare with different types', async () => {
    const { program, variants } = await compileModels(`
      @service namespace Test;

      model OwnerRef {
        id: string;
      }

      model Owner {
        id: int32;
        display_name: string;
      }
    `)

    expect(
      conflictingVariantProperty(program, variants('Owner', 'OwnerRef')),
    ).toBe('"id" (different types in Owner and OwnerRef)')
  })

  it('compares JSON names, not declared names', async () => {
    const { program, variants } = await compileModels(`
      @service namespace Test;

      model OwnerRef {
        @encodedName("application/json", "owner_id")
        id: string;
      }

      model Owner {
        @encodedName("application/json", "owner_id")
        ownerId: int32;
      }
    `)

    expect(
      conflictingVariantProperty(program, variants('Owner', 'OwnerRef')),
    ).toBe('"owner_id" (different types in Owner and OwnerRef)')
  })
})
