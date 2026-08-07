import type { Model, Program, Union } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { describe, expect, it } from 'vitest'
import {
  conflictingVariantProperty,
  variantGoName,
} from '../dist/components/GoUnion.js'

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

describe('union variant accessor naming', () => {
  async function compileUnion(code: string): Promise<{
    program: Program
    union: (name: string) => Union
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
    const unions = runner.program
      .getGlobalNamespaceType()
      .namespaces.get('Test')!.unions
    return {
      program: runner.program,
      union: (name) => unions.get(name)!,
    }
  }

  it('names a union-typed variant after the nested union, not the placeholder fallback', async () => {
    const { program, union } = await compileUnion(`
      @service namespace Test;

      model InvoiceStandard {
        type: "standard";
        id: string;
      }

      @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
      union Invoice {
        standard: InvoiceStandard,
      }

      model InvoiceReference {
        id: string;
      }

      union InvoiceOrReference {
        Invoice,
        InvoiceReference,
      }
    `)

    const names = [...union('InvoiceOrReference').variants.values()].map(
      (variant) => variantGoName(program, 'InvoiceOrReference', variant),
    )
    expect(names).toEqual(['Invoice', 'InvoiceReference'])
  })
})
