import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";
using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
  model Options { as_of?: utcDateTime; }
  interface Operations {
    @post @route("/{id}/optional") @operationId("optional-widget")
    optional(@path id: string, @body request?: Options): void;
    @post @route("/optional") @operationId("optional-widgets")
    optionalOnly(@body request?: Options): void;
    @post @route("/optional-query") @operationId("optional-query-widgets")
    optionalQuery(@query q?: string, @body request?: Options): void;
    @post @route("/required") @operationId("required-widgets")
    required(@body request: Options): void;
  }
}
@service namespace Api {
  @route("/widgets") interface Endpoints extends Widgets.Operations {}
}
`

describe('optional request bodies', () => {
  let outputs: Record<string, string>

  beforeAll(async () => {
    const [result, diagnostics] =
      await EmitterTester.compileAndDiagnose(FIXTURE)
    expect(diagnostics.filter((d) => d.severity === 'error')).toEqual([])
    outputs = result.outputs
  })

  it('preserves optional bodies alongside required path parameters', () => {
    expect(outputs['src/models/operations/widgets.ts']).toContain(
      'id: string\n  body?: Options',
    )
    expect(outputs['src/funcs/widgets.ts']).toContain(
      'client._options.validate && body !== undefined',
    )
  })

  it('allows omitting a body-only request but keeps required bodies required', () => {
    const operations = outputs['src/models/operations/widgets.ts']!
    expect(operations).toContain(
      'OptionalWidgetsRequest = AcceptDateStrings<Options> | undefined',
    )
    expect(operations).toContain(
      'RequiredWidgetsRequest = AcceptDateStrings<Options>\n',
    )
    const funcs = outputs['src/funcs/widgets.ts']!
    expect(funcs).toContain('req?: OptionalWidgetsRequest')
    expect(funcs).toContain('req: RequiredWidgetsRequest,')
    expect(funcs).toContain(
      'if (client._options.validate) {\n      assertValid(schemas.requiredWidgetsBodyWire, body)',
    )
    expect(outputs['src/sdk/widgets.ts']).toContain(
      'request?: OptionalWidgetsRequest',
    )
  })

  it('keeps a query-bearing request required while its body stays optional', () => {
    expect(outputs['src/models/operations/widgets.ts']).toContain(
      'OptionalQueryWidgetsRequest = AcceptDateStrings<{\n  body?: Options\n} & OptionalQueryWidgetsQuery>',
    )
    const funcs = outputs['src/funcs/widgets.ts']!
    expect(funcs).toContain('req: OptionalQueryWidgetsRequest,')
    expect(funcs).toContain(
      'toWire(req.body, schemas.optionalQueryWidgetsBody)',
    )
    expect(funcs).toContain(
      'if (client._options.validate && body !== undefined) {\n      assertValid(schemas.optionalQueryWidgetsBodyWire, body)',
    )
    expect(outputs['src/sdk/widgets.ts']).toContain(
      'request: OptionalQueryWidgetsRequest',
    )
  })
})
