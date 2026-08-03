import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
  scalar WidgetId extends string;

  model Widget {
    id: WidgetId;
  }

  interface WidgetOperations {
    @get
    @route("/{widgetId}")
    @operationId("get-widget")
    get(@path widgetId: WidgetId): Widget;
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/widgets")
  interface WidgetEndpoints extends Widgets.WidgetOperations {}
}
`

describe('path parameter validation', () => {
  let funcs: string

  beforeAll(async () => {
    const [result, diagnostics] =
      await EmitterTester.compileAndDiagnose(FIXTURE)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    funcs = result.outputs['src/funcs/widgets.ts']!
  })

  it('validates mapped path values before URL interpolation', () => {
    const mapping = funcs.indexOf(
      'toPathWire(pathParamsInput, schemas.getWidgetPathParams)',
    )
    const validation = funcs.indexOf(
      'assertValid(schemas.getWidgetPathParamsWire, pathParams)',
    )
    const interpolation = funcs.indexOf('const path = `widgets/${')

    expect(mapping).toBeGreaterThan(-1)
    expect(validation).toBeGreaterThan(mapping)
    expect(interpolation).toBeGreaterThan(validation)
  })

  it('keeps mapping conditional on strict validation', () => {
    expect(funcs).toContain(
      'const pathParams = client._options.validate\n      ? toPathWire(',
    )
    expect(funcs).toContain(': pathParamsInput')
  })
})
