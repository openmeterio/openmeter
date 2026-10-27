import { describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

describe('documented array responses', () => {
  it('references the documented model for anonymous and named arrays', async () => {
    const [result, diagnostics] = await EmitterTester.compileAndDiagnose(`
      import "@typespec/http";
      import "@typespec/openapi";
      using TypeSpec.Http;
      using TypeSpec.OpenAPI;

      namespace Widgets {
        model Widget { @doc("The widget identifier.") id: string; }
        model WidgetArray is Array<Widget>;
        interface Operations {
          @get @operationId("list-widgets") list(): Widget[];
          @get @route("/named") @operationId("named-widgets") named(): WidgetArray;
        }
      }
      @service namespace Api {
        @route("/widgets") interface Endpoints extends Widgets.Operations {}
      }
    `)
    expect(diagnostics.filter((d) => d.severity === 'error')).toEqual([])
    const operations = result.outputs['src/models/operations/widgets.ts']!
    expect(operations).toContain('ListWidgetsResponse = Widget[]')
    expect(operations).toContain('NamedWidgetsResponse = WidgetArray')
    expect(result.outputs['src/models/types.ts']).toContain(
      'WidgetArray = Widget[]',
    )
    expect(operations).toMatch(/import type \{[^}]*Widget[^}]*\} from/)
    expect(operations).not.toContain('z.output')
    expect(result.outputs['src/models/types.ts']).toContain(
      'The widget identifier.',
    )
  })
})
