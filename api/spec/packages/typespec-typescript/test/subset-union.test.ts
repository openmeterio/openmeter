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

// A reference model whose properties are a subset of the full model's — the
// shape `BillingChargesCustomerOrReference` has in the real spec.
const SUBSET = spec(`
  model OwnerRef {
    id: string;
  }

  model Owner {
    id: string;
    display_name: string;
  }

  union OwnerOrRef {
    owner: Owner,
    ref: OwnerRef,
  }
`)

// The same two variants, but `id` means something different in each: the
// mapper's pick would decide how `id` maps, which it cannot do.
const CONFLICTING = spec(`
  model OwnerRef {
    id: string;
  }

  model Owner {
    id: int32;
    display_name: string;
  }

  union OwnerOrRef {
    owner: Owner,
    ref: OwnerRef,
  }
`)

const ownerRef = z.object({ id: z.string() })
const owner = z.object({ id: z.string(), displayName: z.string() })
const payload = z.object({ owner: z.union([owner, ownerRef]) })

describe('non-discriminated union of subset variants', () => {
  let schemas: string

  beforeAll(async () => {
    const [result, diagnostics] = await EmitterTester.compileAndDiagnose(SUBSET)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    schemas = result.outputs['src/models/schemas.ts']!
  })

  it('emits the union instead of failing the casing gate', () => {
    expect(schemas).toMatch(/ownerOrRef = z\.union\(\[owner, ownerRef\]\)/)
    expect(schemas).toMatch(
      /ownerOrRefWire = z\.union\(\[ownerWire, ownerRefWire\]\)/,
    )
  })

  it('maps the wide variant with its own keys', () => {
    const wire = { owner: { id: 'o1', display_name: 'Acme' } }
    const publicValue = { owner: { id: 'o1', displayName: 'Acme' } }

    expect(fromWire(wire, payload)).toEqual(publicValue)
    expect(toWire(publicValue, payload)).toEqual(wire)
  })

  it('maps the narrow variant without inventing the wide variant keys', () => {
    const wire = { owner: { id: 'o1' } }
    const publicValue = { owner: { id: 'o1' } }

    expect(fromWire(wire, payload)).toEqual(publicValue)
    expect(toWire(publicValue, payload)).toEqual(wire)
  })

  it('does not apply a wider variant default to narrow-variant data', () => {
    const wide = z.object({ id: z.string(), kind: z.string().default('full') })
    const body = z.object({ owner: z.union([wide, ownerRef]) })

    expect(toWire({ owner: { id: 'o1' } }, body)).toEqual({
      owner: { id: 'o1' },
    })
    expect(toWire({ owner: { id: 'o1', kind: 'full' } }, body)).toEqual({
      owner: { id: 'o1', kind: 'full' },
    })
  })

  it('still fails the build when variants disagree on a shared key', async () => {
    await expect(EmitterTester.compileAndDiagnose(CONFLICTING)).rejects.toThrow(
      /'id' has a different type in Owner and OwnerRef/,
    )
  })
})
