import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Common {
  model SortQuery {
    by: string;
    order?: "asc" | "desc" = "asc";
  }
}

namespace Widgets {
  model Widget {
    id: string;
  }

  interface WidgetOperations {
    @get
    @operationId("list-widgets")
    list(@query(#{ name: "sort" }) sort?: Common.SortQuery): Widget[];

    @get
    @route("/raw-sort")
    @operationId("search-widgets")
    search(@query(#{ name: "sort" }) sort?: string): Widget[];
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/widgets")
  interface WidgetEndpoints extends Widgets.WidgetOperations {}
}
`

describe('sort query validation', () => {
  let outputs: Record<string, string>

  const file = (path: string): string => {
    const content = outputs[path]
    expect(content, `expected emitted file ${path}`).toBeDefined()
    return content!
  }

  beforeAll(async () => {
    const [result, diagnostics] =
      await EmitterTester.compileAndDiagnose(FIXTURE)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    outputs = result.outputs
  })

  it('keeps the ergonomic public object but validates its encoded wire string', () => {
    const schemas = file('src/models/schemas.ts')
    expect(schemas).toMatch(/export const sortQuery = z\s*\.object\(/)
    expect(schemas).toMatch(/export const sortQueryWire = z\s*\.string\(\)/)
    expect(schemas).toMatch(
      /export const listWidgetsQueryParamsWire = z\.object\(\{\s*sort: sortQueryWire\.optional\(\)/,
    )
  })

  it('validates SortQuery before encoding and the effective query afterward', () => {
    const funcs = file('src/funcs/widgets.ts')
    const publicValidation = funcs.indexOf(
      'assertValid(schemas.sortQuery, req.sort)',
    )
    const encoding = funcs.indexOf('sort: encodeSort(req.sort, toSnakeCase)')
    const wireValidation = funcs.indexOf(
      'assertValid(schemas.listWidgetsQueryParamsWire, query)',
    )

    expect(publicValidation).toBeGreaterThan(-1)
    expect(encoding).toBeGreaterThan(publicValidation)
    expect(wireValidation).toBeGreaterThan(encoding)
  })

  it('does not apply the object codec to an unrelated scalar named sort', () => {
    const funcs = file('src/funcs/widgets.ts')
    const search = funcs.slice(funcs.indexOf('export function searchWidgets('))
    expect(search).toContain('sort: req.sort')
    expect(search).not.toContain('encodeSort(req.sort')
  })
})
