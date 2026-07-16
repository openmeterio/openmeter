import { beforeAll, describe, expect, it } from 'vitest'
import { z } from 'zod'
import { fromWire, toWire } from '../src/runtime/wire.js'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
  model FlatCost {
    type: "flat";
    amount: int32;
  }

  model DynamicCost {
    type: "dynamic";
    provider_property: string;
    model_property: string;
  }

  @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
  union Cost {
    flat: FlatCost,
    dynamic: DynamicCost,
  }

  model Payload {
    nullable_cost?: Cost | null;
    multi: "alpha" | 42 | null;
    null_only: null;
    plain_union: "alpha" | 42;
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

describe('nullable union emission', () => {
  let schemas: string

  beforeAll(async () => {
    const [result, diagnostics] =
      await EmitterTester.compileAndDiagnose(FIXTURE)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    schemas = result.outputs['src/models/schemas.ts']!
  })

  it('preserves a nullable discriminated union in public and wire schemas', () => {
    expect(schemas).toMatch(/nullableCost: cost\.nullable\(\)\.optional\(\)/)
    expect(schemas).toMatch(
      /nullable_cost: costWire\.nullable\(\)\.optional\(\)/,
    )
    expect(schemas).not.toMatch(
      /z\.union\(\[\s*cost(?:Wire)?,\s*z\.null\(\)\s*\]\)/,
    )
  })

  it('keeps every non-null variant in a multi-variant nullable union', () => {
    expect(
      schemas.match(
        /multi: z\s*\.union\(\[z\.literal\("alpha"\), z\.literal\(42\)\]\)\s*\.nullable\(\)/g,
      ),
    ).toHaveLength(2)
  })

  it('keeps bare null intrinsic and non-null union emission unchanged', () => {
    expect(schemas.match(/nullOnly: z\.null\(\)/g)).toHaveLength(1)
    expect(schemas.match(/null_only: z\.null\(\)/g)).toHaveLength(1)
    expect(schemas).toMatch(
      /plainUnion: z\s*\.union\(\[z\.literal\("alpha"\), z\.literal\(42\)\]\)(?!\s*\.nullable\(\))/,
    )
    expect(schemas).toMatch(
      /plain_union: z\s*\.union\(\[z\.literal\("alpha"\), z\.literal\(42\)\]\)(?!\s*\.nullable\(\))/,
    )
  })

  it('round-trips renamed keys through a nullable discriminated union', () => {
    const cost = z.discriminatedUnion('type', [
      z.object({ type: z.literal('flat'), amount: z.number() }),
      z.object({
        type: z.literal('dynamic'),
        providerProperty: z.string(),
        modelProperty: z.string(),
      }),
    ])
    const body = z.object({ nullableCost: cost.nullable().optional() })
    const bodyWire = z.strictObject({
      nullable_cost: z
        .discriminatedUnion('type', [
          z.strictObject({ type: z.literal('flat'), amount: z.number() }),
          z.strictObject({
            type: z.literal('dynamic'),
            provider_property: z.string(),
            model_property: z.string(),
          }),
        ])
        .nullable()
        .optional(),
    })
    const input = {
      nullableCost: {
        type: 'dynamic' as const,
        providerProperty: 'provider',
        modelProperty: 'model',
      },
    }

    const wire = toWire(input, body)

    expect(body.safeParse(input).success).toBe(true)
    expect(wire).toEqual({
      nullable_cost: {
        type: 'dynamic',
        provider_property: 'provider',
        model_property: 'model',
      },
    })
    expect(bodyWire.safeParse(wire).success).toBe(true)
    expect(fromWire(wire, body)).toEqual(input)

    const nullWire = toWire({ nullableCost: null }, body)
    expect(nullWire).toEqual({ nullable_cost: null })
    expect(bodyWire.safeParse(nullWire).success).toBe(true)
    expect(fromWire(nullWire, body)).toEqual({ nullableCost: null })
  })
})
