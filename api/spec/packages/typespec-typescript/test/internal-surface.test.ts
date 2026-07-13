import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

// A minimal spec exercising the customer-visibility markers, authored with the
// same extends pattern the real spec uses (grouping resolves through the
// source interface's namespace). Widgets mixes public and internal operations;
// Gadgets is entirely internal; delete-widget is private and update-widget
// carries both markers (private wins).
const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Shared {
  model PagePaginatedResponse<T> {
    data: T[];
    meta: PageMeta;
  }
  model PageMeta {
    page: PageInfo;
  }
  model PageInfo {
    number: int32;
    size: int32;
    total: int32;
  }
}

namespace Widgets {
  model Widget {
    id: string;
    name: string;
  }

  interface WidgetOperations {
    @get
    @operationId("list-widgets")
    list(): Shared.PagePaginatedResponse<Widget>;

    @post
    @operationId("create-widget")
    @extension("x-internal", true)
    create(@body body: Widget): Widget;

    @delete
    @operationId("delete-widget")
    @extension("x-private", true)
    delete(@path widgetId: string): void;

    @put
    @operationId("update-widget")
    @extension("x-internal", true)
    @extension("x-private", true)
    update(@path widgetId: string, @body body: Widget): Widget;
  }
}

namespace Gadgets {
  model Gadget {
    id: string;
  }

  interface GadgetOperations {
    @get
    @operationId("list-gadgets")
    @extension("x-internal", true)
    list(): Shared.PagePaginatedResponse<Gadget>;
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/widgets")
  interface WidgetEndpoints extends Widgets.WidgetOperations {}

  @route("/gadgets")
  interface GadgetEndpoints extends Gadgets.GadgetOperations {}
}
`

describe('internal operation surface', () => {
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

  it('drops x-private operations entirely, even when also marked x-internal', () => {
    for (const content of Object.values(outputs)) {
      expect(content).not.toContain('deleteWidget')
      expect(content).not.toContain('updateWidget')
    }
  })

  it('keeps internal operations out of the public facade', () => {
    const widgets = file('src/sdk/widgets.ts')
    expect(widgets).toContain('export class Widgets {')
    expect(widgets).toContain('async list(')
    expect(widgets).toContain('listAll(')
    expect(widgets).not.toContain('create')
  })

  it('quarantines internal operations under Internal<Group> facades', () => {
    const internal = file('src/sdk/internal.ts')
    expect(internal).toContain('export class Internal {')
    expect(internal).toContain('get widgets(): InternalWidgets {')
    expect(internal).toContain('get gadgets(): InternalGadgets {')
    expect(internal).toContain('export class InternalWidgets {')
    expect(internal).toContain('async create(')
    expect(internal).toContain('export class InternalGadgets {')
    expect(internal).toContain('async list(')
    // Pagination companions are emitted for internal operations too.
    expect(internal).toContain('listAll(')
  })

  it('emits no public facade or getter for an entirely-internal group', () => {
    expect(outputs['src/sdk/gadgets.ts']).toBeUndefined()
    const sdk = file('src/sdk/sdk.ts')
    expect(sdk).not.toContain('get gadgets')
    const index = file('src/index.ts')
    expect(index).not.toContain('export { Gadgets }')
  })

  it('exposes the internal aggregate through the client but not the package root', () => {
    const sdk = file('src/sdk/sdk.ts')
    expect(sdk).toContain("import { Internal } from './internal.js'")
    expect(sdk).toContain('get internal(): Internal {')
    const index = file('src/index.ts')
    expect(index).toContain("export { Widgets } from './sdk/widgets.js'")
    expect(index).not.toContain('export { Internal }')
  })

  it('shares funcs and operation envelope modules between surfaces', () => {
    expect(file('src/funcs/widgets.ts')).toContain(
      'export function createWidget(',
    )
    expect(file('src/funcs/gadgets.ts')).toContain(
      'export function listGadgets(',
    )
    expect(file('src/funcs/index.ts')).toContain("export * from './gadgets.js'")
    expect(file('src/models/operations/widgets.ts')).toContain(
      'CreateWidgetRequest',
    )
    const index = file('src/index.ts')
    expect(index).toContain(
      "export type * from './models/operations/gadgets.js'",
    )
  })

  it('documents the internal surface in a dedicated README section', () => {
    const readme = file('README.md')
    expect(readme).toContain('## Internal Operations')
    expect(readme).toContain('`client.internal.widgets.create`')
    expect(readme).toContain('### Internal Gadgets')
    expect(readme).toContain('`client.internal.gadgets.list`')
    // The public table lists only the public operation.
    expect(readme).toContain('`client.widgets.list`')
    expect(readme).not.toContain('`client.widgets.create`')
  })
})
