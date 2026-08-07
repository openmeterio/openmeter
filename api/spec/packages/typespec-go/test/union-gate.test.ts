import type { Model, Program, Union } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { describe, expect, it } from 'vitest'
import {
  conflictingVariantProperty,
  nestedUnionModelVariants,
  variantGoName,
} from '../dist/components/GoUnion.js'

async function compileModels(code: string): Promise<{
  program: Program
  variants: (...names: string[]) => Model[]
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
  const namespace = runner.program
    .getGlobalNamespaceType()
    .namespaces.get('Test')!
  return {
    program: runner.program,
    variants: (...names) => names.map((name) => namespace.models.get(name)!),
    union: (name) => namespace.unions.get(name)!,
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

// The expandable-reference shape: `Invoice` (a discriminated union of its
// concrete types) or a bare `InvoiceReference`. Both compete for the same
// payload in the As* accessors, so the nested union's models have to agree with
// the outer reference on every JSON key they share.
const NESTED_UNION = `
  @service namespace Test;

  model InvoiceStandard {
    type: "standard";
    id: string;
    amount: string;
  }

  model InvoiceCredit {
    type: "credit";
    id: string;
    amount: int32;
  }

  @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
  union Invoice {
    standard: InvoiceStandard,
    credit: InvoiceCredit,
  }

  model InvoiceReference {
    id: string;
  }

  model StrictInvoiceReference {
    id: int32;
  }

  union InvoiceOrReference {
    Invoice,
    InvoiceReference,
  }
`

describe('nested union variant compatibility', () => {
  it('collects the models a union-typed variant contributes', async () => {
    const { union } = await compileModels(NESTED_UNION)

    expect(
      nestedUnionModelVariants(union('InvoiceOrReference')).map((m) => m.name),
    ).toEqual(['InvoiceStandard', 'InvoiceCredit'])
  })

  it('reports a nested variant that disagrees with an outer sibling', async () => {
    const { program, variants, union } = await compileModels(NESTED_UNION)

    expect(
      conflictingVariantProperty(
        program,
        variants('StrictInvoiceReference'),
        nestedUnionModelVariants(union('InvoiceOrReference')),
      ),
    ).toBe(
      '"id" (different types in StrictInvoiceReference and InvoiceStandard)',
    )
  })

  // The nested union selects on its discriminator, so its own variants never
  // have to be told apart by shape. Comparing them with each other would fail
  // the build on unions that decode correctly.
  it("ignores disagreements among the nested union's own variants", async () => {
    const { program, variants, union } = await compileModels(NESTED_UNION)

    expect(
      conflictingVariantProperty(
        program,
        variants('InvoiceReference'),
        nestedUnionModelVariants(union('InvoiceOrReference')),
      ),
    ).toBeUndefined()
  })
})

describe('union variant accessor naming', () => {
  it('names a union-typed variant after the nested union, not the placeholder fallback', async () => {
    const { program, union } = await compileModels(`
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
