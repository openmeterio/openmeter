import { beforeAll, describe, expect, it } from 'vitest'
import { z } from 'zod'
import { fromWire, toWire } from '../src/runtime/wire.js'
import { EmitterTester } from './emit.js'

// The expandable-reference shape whose expanded side is itself a union: a
// charge realization's `invoice` is either the full `Invoice` (a discriminated
// union of its concrete types) or a bare `InvoiceReference`. The outer union is
// therefore a union of a union and an object.
const NESTED = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
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
})
