import { beforeAll, describe, expect, it } from 'vitest'
import { z } from 'zod'
import { fromWire, toWire } from '../src/runtime/wire.js'
import { EmitterTester } from './emit.js'

const spec = (types: string) => `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
${types}

  model Payload {
    owner: OwnerOrRef;
  }

  interface WidgetOperations {
    @post
    @operationId("create-widget")
    create(@body body: Payload): Payload;
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/widgets")
  interface WidgetEndpoints extends Widgets.WidgetOperations {}
}
`

// The expandable-reference shape whose expanded side is itself a union: a
// charge realization's `invoice` is either the full `Invoice` (a discriminated
// union of its concrete types) or a bare `InvoiceReference`. The outer union is
// therefore a union of a union and an object.
const NESTED = spec(`
  model OwnerRef {
    id: string;
  }

  model StandardOwner {
    type: "standard";
    id: string;
    display_name: string;
  }

  @discriminated(#{ discriminatorPropertyName: "type", envelope: "none" })
  union Owner {
    standard: StandardOwner,
  }

  union OwnerOrRef {
    owner: Owner,
    ref: OwnerRef,
  }
`)

// The nested union's variant disagrees with the OUTER reference variant on `id`.
// Both compete in the same key-coverage pick, so the mapper's choice would decide
// how `id` maps — exactly what the gate exists to prevent.
const NESTED_CONFLICT = spec(`
  model OwnerRef {
    id: string;
  }

  model StandardOwner {
    type: "standard";
    id: int32;
    display_name: string;
  }

  @discriminated(#{ discriminatorPropertyName: "type", envelope: "none" })
  union Owner {
    standard: StandardOwner,
  }

  union OwnerOrRef {
    owner: Owner,
    ref: OwnerRef,
  }
`)

// The nested union's own variants disagree on `amount`, and neither disagrees
// with the outer reference. Legal: the nested union dispatches on its
// discriminator, so it never has to tell the two apart by shape.
const NESTED_SIBLINGS_DIFFER = spec(`
  model OwnerRef {
    id: string;
  }

  model StandardOwner {
    type: "standard";
    id: string;
    amount: string;
  }

  model CreditOwner {
    type: "credit";
    id: string;
    amount: int32;
  }

  @discriminated(#{ discriminatorPropertyName: "type", envelope: "none" })
  union Owner {
    standard: StandardOwner,
    credit: CreditOwner,
  }

  union OwnerOrRef {
    owner: Owner,
    ref: OwnerRef,
  }
`)

const ownerRef = z.object({ id: z.string() })
const owner = z.discriminatedUnion('type', [
  z.object({
    type: z.literal('standard'),
    id: z.string(),
    displayName: z.string(),
  }),
])
const payload = z.object({ owner: z.union([owner, ownerRef]) })

describe('non-discriminated union with a union variant', () => {
  let schemas: string

  beforeAll(async () => {
    const [result, diagnostics] = await EmitterTester.compileAndDiagnose(NESTED)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    schemas = result.outputs['src/models/schemas.ts']!
  })

  it('emits the union of a union and an object', () => {
    expect(schemas).toMatch(/ownerOrRef = z\.union\(\[owner, ownerRef\]\)/)
    expect(schemas).toMatch(
      /ownerOrRefWire = z\.union\(\[ownerWire, ownerRefWire\]\)/,
    )
  })

  // The regression: the nested union is no object, so a mapper that only
  // considers object options skips it entirely, leaving the reference the sole
  // candidate and silently dropping every field of the expanded resource.
  it('maps the expanded variant through the nested union', () => {
    const wire = {
      owner: { type: 'standard', id: 'o1', display_name: 'Acme' },
    }
    const publicValue = {
      owner: { type: 'standard', id: 'o1', displayName: 'Acme' },
    }

    expect(fromWire(wire, payload)).toEqual(publicValue)
    expect(toWire(publicValue, payload)).toEqual(wire)
  })

  it('still maps the reference variant, which carries no discriminator', () => {
    const wire = { owner: { id: 'o1' } }

    expect(fromWire(wire, payload)).toEqual(wire)
    expect(toWire(wire, payload)).toEqual(wire)
  })

  it('fails the build when a nested variant disagrees with an outer sibling', async () => {
    await expect(
      EmitterTester.compileAndDiagnose(NESTED_CONFLICT),
    ).rejects.toThrow(/'id' has a different type in OwnerRef and StandardOwner/)
  })

  it('allows the nested union to disagree with itself on a non-discriminator key', async () => {
    const [, diagnostics] = await EmitterTester.compileAndDiagnose(
      NESTED_SIBLINGS_DIFFER,
    )
    expect(diagnostics.filter((d) => d.severity === 'error')).toEqual([])
  })
})
