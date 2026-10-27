import { describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

describe('response name collisions', () => {
  it('aliases output models that collide with generated response names', async () => {
    const [result, diagnostics] = await EmitterTester.compileAndDiagnose(`
      import "@typespec/http";
      import "@typespec/openapi";
      using TypeSpec.Http;
      using TypeSpec.OpenAPI;

      namespace Widgets {
        model GetWidgetResponse { id: string; }
        interface Operations {
          @get @operationId("get-widget") get(): GetWidgetResponse;
        }
      }
      @service namespace Api {
        @route("/widgets") interface Endpoints extends Widgets.Operations {}
      }
    `)
    expect(diagnostics.filter((d) => d.severity === 'error')).toEqual([])
    const operations = result.outputs['src/models/operations/widgets.ts']!
    expect(operations).toContain('GetWidgetResponse as GetWidgetResponseBody')
    expect(operations).toContain('GetWidgetResponse = GetWidgetResponseBody')
  })
})
