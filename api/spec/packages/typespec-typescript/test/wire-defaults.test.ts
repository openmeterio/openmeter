import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Widgets {
  model Widget {
    id: string;
    kind: string = "standard";
    note?: string = "server-default";
  }

  interface WidgetOperations {
    @post
    @operationId("create-widget")
    create(@body body: Widget): Widget;
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/widgets")
  interface WidgetEndpoints extends Widgets.WidgetOperations {}
}
`

describe('wire defaults', () => {
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

  it('keeps defaults on the public schema for request materialization', () => {
    const publicSchema = schemas.slice(
      schemas.indexOf('export const widget ='),
      schemas.indexOf('export const widgetWire ='),
    )
    expect(publicSchema).toContain('.default("standard")')
    expect(publicSchema).toContain('.optional().default("server-default")')
  })

  it('requires required fields and only preserves optionality on the wire', () => {
    const wireSchema = schemas.slice(
      schemas.indexOf('export const widgetWire ='),
    )
    expect(wireSchema).not.toContain('.default(')
    expect(wireSchema).toMatch(/kind: z\.string\(\)/)
    expect(wireSchema).toMatch(/note: z\.string\(\)\.optional\(\)/)
  })
})
