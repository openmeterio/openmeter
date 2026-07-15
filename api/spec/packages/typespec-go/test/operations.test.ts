import type { Model, Program } from '@typespec/compiler'
import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { describe, expect, it } from 'vitest'
import {
  collectHttpOperations,
  describeOperations,
  jsonBodyOverrides,
  type GoOperation,
} from '../dist/operations.js'
import { groupOperations, operationNestPath } from '../dist/grouping.js'
import { computeDivergentTypes } from '../dist/projections.js'
import { validateUniqueTypeNames } from '../dist/emitter.js'
import { isRuntimeBackedTypeName } from '../dist/runtime-symbols.js'
import {
  configureGoTypeNames,
  optionalTypeName,
  resolveGoTypeNames,
} from '../dist/go-types.js'

async function compileProgram(code: string): Promise<Program> {
  const host = await createTestHost({
    libraries: [HttpTestLibrary, OpenAPITestLibrary],
  })
  const runner = await createTestRunner(host)
  await runner.compile(`
    import "@typespec/http";
    import "@typespec/openapi";
    using TypeSpec.Http;
    using TypeSpec.OpenAPI;
    ${code}
  `)

  return runner.program
}

async function compileOperations(code: string): Promise<GoOperation[]> {
  const program = await compileProgram(code)
  const operations = collectHttpOperations(program)
  return describeOperations(program, 'Test', operations)
}

describe('operation HTTP IR', () => {
  it('classifies every supported query encoding from HTTP metadata', async () => {
    const operations = await compileOperations(`
      @service namespace Test;

      namespace Common {
        model SortQuery {
          by: string;
          order?: "asc" | "desc";
        }
      }

      model ItemFilter {
        key?: string;
      }

      @route("/items")
      interface Operations {
        @get
        op list(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
          @query sort?: Common.SortQuery,
          @query(#{ style: "deepObject", explode: true }) filter?: ItemFilter,
          @query(#{ explode: true }) expand?: string[],
          @query timestamp?: utcDateTime,
        ): string[];

        @get
        @route("/cursor")
        op cursor(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            after?: string;
            before?: string;
          },
        ): string[];
      }
    `)

    const list = operations.find(
      (operation) => operation.operation.name === 'list',
    )!
    expect(
      Object.fromEntries(
        list.queryParams.map((parameter) => [
          parameter.name,
          parameter.queryCodec,
        ]),
      ),
    ).toMatchObject({
      page: { kind: 'page' },
      sort: { kind: 'sort' },
      filter: { kind: 'deepObject' },
      expand: { kind: 'array', explode: true },
      timestamp: { kind: 'scalar' },
    })
    expect(list.pagination).toBe('page')

    const cursor = operations.find(
      (operation) => operation.operation.name === 'cursor',
    )!
    expect(cursor.queryParams[0]?.queryCodec).toEqual({
      kind: 'cursorPage',
    })
    expect(cursor.pagination).toBe('cursor')
  })

  it('retains optional body and media-type metadata', async () => {
    const operations = await compileOperations(`
      @service namespace Test;

      model Request {
        value: string;
      }

      model Response {
        value: string;
      }

      @route("/items")
      interface Operations {
        @post
        op create(@body body?: Request): Response;
      }
    `)

    expect(operations).toHaveLength(1)
    expect(operations[0]).toMatchObject({
      bodyOptional: true,
      requestContentType: 'application/json',
      responseContentType: 'application/json',
    })
  })

  it('retains shared-route media-type variants as distinct methods', async () => {
    const operations = await compileOperations(`
      @service namespace Test;

      model Event {
        id: string;
      }

      model Response {
        accepted: boolean;
      }

      @route("/events")
      interface Operations {
        @post
        @operationId("ingest-metering-events")
        @sharedRoute
        ingestEvent(
          @header contentType: "application/cloudevents+json",
          @body body: Event,
        ): Response;

        @post
        @operationId("ingest-metering-events")
        @sharedRoute
        ingestEvents(
          @header contentType: "application/cloudevents-batch+json",
          @body body: Event[],
        ): Response;

        @post
        @operationId("ingest-metering-events")
        @sharedRoute
        ingestEventsJson(
          @header contentType: "application/json",
          @body body: Event | Event[],
        ): Response;
      }
    `)

    expect(operations.map((operation) => operation.methodName)).toEqual([
      'IngestEvent',
      'IngestEvents',
      'IngestEventsJSON',
    ])
    expect(operations.map((operation) => operation.requestContentType)).toEqual(
      [
        'application/cloudevents+json',
        'application/cloudevents-batch+json',
        'application/json',
      ],
    )
    expect(operations.map((operation) => operation.body?.kind)).toEqual([
      'Model',
      'Model',
      'Union',
    ])
  })

  it('rejects operations that cannot be traced to a resource namespace', async () => {
    const program = await compileProgram(`
      @service namespace Test;

      @route("/items")
      interface Operations {
        @get op list(): string[];
      }
    `)

    const operations = collectHttpOperations(program)
    expect(() => groupOperations(operations)).toThrow(
      'cannot place operation Operations.list',
    )
  })

  it('groups operations by the namespace of their source interface', async () => {
    const program = await compileProgram(`
      namespace Widgets {
        interface Operations {
          @get op list(): string[];
        }
      }

      @service
      namespace Test {
        @route("/widgets")
        interface Endpoints extends Widgets.Operations {}
      }
    `)

    const operations = collectHttpOperations(program)
    expect([...groupOperations(operations).keys()]).toEqual(['Widgets'])
  })

  it('keeps same-named operations in different containers from sharing bodies', async () => {
    const program = await compileProgram(`
      @service namespace Test;

      model Payload {
        value: string;
      }

      @route("/widgets")
      interface Widgets {
        @get op list(): string[];
      }

      @route("/gadgets")
      interface Gadgets {
        @post op list(@body body: Payload): string[];
      }
    `)

    const overrides = jsonBodyOverrides(program)
    expect(overrides.size).toBe(0)

    const operations = describeOperations(
      program,
      'Test',
      collectHttpOperations(program),
      overrides,
    )
    const widgetsList = operations.find(
      (operation) => operation.path === '/widgets',
    )!
    expect(widgetsList.body).toBeUndefined()
    const gadgetsList = operations.find(
      (operation) => operation.path === '/gadgets',
    )!
    expect((gadgetsList.body as Model | undefined)?.name).toBe('Payload')
  })

  it('attaches the shared-route JSON body to bodyless siblings by qualified key', async () => {
    const program = await compileProgram(`
      @service namespace Test;

      model QueryRequest {
        value: string;
      }

      model QueryResult {
        value: string;
      }

      @route("/query")
      interface Operations {
        @post
        @operationId("query-thing")
        @sharedRoute
        query(@body request: QueryRequest): {
          @header contentType: "application/json";
          @body _: QueryResult;
        };

        @friendlyName("queryThingCsv")
        @post
        @operationId("query-thing")
        @sharedRoute
        queryCsv(): {
          @header contentType: "text/csv";
          @body _: string;
        };
      }
    `)

    const overrides = jsonBodyOverrides(program)
    expect([...overrides.keys()]).toEqual(['Test.Operations.queryCsv'])

    const operations = describeOperations(
      program,
      'Test',
      collectHttpOperations(program),
      overrides,
    )
    const csv = operations.find(
      (operation) => operation.operation.name === 'queryCsv',
    )!
    expect(csv.requestContentType).toBe('application/json')
    expect((csv.body as Model | undefined)?.name).toBe('QueryRequest')
  })

  it('rejects multiple 2xx response bodies with different types', async () => {
    await expect(
      compileOperations(`
        @service namespace Test;

        model Created {
          id: string;
        }

        model Accepted {
          token: string;
        }

        @route("/items")
        interface Operations {
          @post op create(): {
            @statusCode _: 201;
            @body body: Created;
          } | {
            @statusCode _: 202;
            @body body: Accepted;
          };
        }
      `),
    ).rejects.toThrow('multiple 2xx response bodies with different types')
  })

  it('accepts multiple 2xx responses sharing one body type', async () => {
    const operations = await compileOperations(`
      @service namespace Test;

      model Created {
        id: string;
      }

      @route("/items")
      interface Operations {
        @post op create(): {
          @statusCode _: 200;
          @body body: Created;
        } | {
          @statusCode _: 201;
          @body body: Created;
        };
      }
    `)

    expect(operations[0]?.response?.kind).toBe('Model')
  })

  it('rejects page parameters that match no pagination shape', async () => {
    await expect(
      compileOperations(`
        @service namespace Test;

        @route("/items")
        interface Operations {
          @get op list(
            @query(#{ style: "deepObject", explode: true })
            page?: {
              size?: integer;
              after?: string;
              tenant?: string;
            },
          ): string[];
        }
      `),
    ).rejects.toThrow('query parameter page on list')

    await expect(
      compileOperations(`
        @service namespace Test;

        @route("/items")
        interface Operations {
          @get op list(
            @query(#{ style: "deepObject", explode: true })
            page?: {
              size?: integer;
            },
          ): string[];
        }
      `),
    ).rejects.toThrow('query parameter page on list')
  })

  it('keeps pagination-shaped models under other names as deep objects', async () => {
    const operations = await compileOperations(`
      @service namespace Test;

      @route("/items")
      interface Operations {
        @get op list(
          @query(#{ style: "deepObject", explode: true })
          filter?: {
            size?: integer;
            after?: string;
          },
        ): string[];
      }
    `)

    expect(operations[0]?.queryParams[0]?.queryCodec).toMatchObject({
      kind: 'deepObject',
    })
    expect(operations[0]?.pagination).toBeUndefined()
  })

  it('rejects caller-controlled headers until a header codec exists', async () => {
    await expect(
      compileOperations(`
        @service namespace Test;

        @route("/items")
        interface Operations {
          @get
          op get(@header requestId?: string): string;
        }
      `),
    ).rejects.toThrow(
      'typespec-go: unsupported header parameter request-id on get',
    )
  })

  it('derives nested service paths from the source namespace', async () => {
    const host = await createTestHost({ libraries: [HttpTestLibrary] })
    const runner = await createTestRunner(host)
    await runner.compile(`
      import "@typespec/http";
      using TypeSpec.Http;

      namespace Customers.Credits.Grants {
        interface Operations {
          @get op list(): string[];
        }
      }

      @service
      namespace Test {
        @route("/grants")
        interface Endpoints extends Customers.Credits.Grants.Operations {}
      }
    `)

    const [operation] = collectHttpOperations(runner.program)
    expect(operationNestPath(operation!, 'Customers')).toEqual([
      'Credits',
      'Grants',
    ])
  })

  it('marks defaulted both-reachable models and their parents as divergent', async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    await runner.compile(`
      model Child {
        mode: string = "default";
      }
      model Parent {
        child: Child;
      }
    `)

    const global = runner.program.getGlobalNamespaceType()
    const child = global.models.get('Child')!
    const parent = global.models.get('Parent')!
    const both = new Set([parent, child])
    const divergent = computeDivergentTypes(runner.program, both, both)

    expect(divergent.has(child)).toBe(true)
    expect(divergent.has(parent)).toBe(true)
  })

  it('marks optional collection both-reachable models as divergent', async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    await runner.compile(`
      model Request {
        labels?: Record<string>;
        features?: string[];
      }
    `)

    const global = runner.program.getGlobalNamespaceType()
    const request = global.models.get('Request')!
    const both = new Set([request])
    const divergent = computeDivergentTypes(runner.program, both, both)

    expect(divergent.has(request)).toBe(true)
  })

  it('keeps request-only models out of the divergent set', async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    await runner.compile(`
      model Request {
        labels?: Record<string>;
      }
    `)

    const global = runner.program.getGlobalNamespaceType()
    const request = global.models.get('Request')!
    const divergent = computeDivergentTypes(
      runner.program,
      new Set(),
      new Set([request]),
    )

    expect(divergent.size).toBe(0)
  })

  it('rejects generated type names that collide with reserved runtime symbols', async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    await runner.compile(`
      model APIError {
        title: string;
      }
    `)

    const apiError = runner.program
      .getGlobalNamespaceType()
      .models.get('APIError')!

    expect(() =>
      validateUniqueTypeNames(runner.program, new Set([apiError])),
    ).toThrow('reserved SDK runtime symbol APIError')
  })

  it('treats SortQuery as a runtime-backed query helper instead of a generated model', () => {
    expect(isRuntimeBackedTypeName('SortQuery', 'Model')).toBe(true)
  })

  it('strips configured Go type-name prefixes only when unambiguous', async () => {
    const host = await createTestHost()
    const runner = await createTestRunner(host)
    await runner.compile(`
      model BillingWidget {
        value: string;
      }

      model MeteringPlan {
        value: string;
      }

      model Plan {
        value: string;
      }
    `)

    const global = runner.program.getGlobalNamespaceType()
    const billingWidget = global.models.get('BillingWidget')!
    const meteringPlan = global.models.get('MeteringPlan')!
    const plan = global.models.get('Plan')!

    configureGoTypeNames(runner.program, ['Billing', 'Metering'])
    resolveGoTypeNames(runner.program, [billingWidget, meteringPlan, plan])

    expect(optionalTypeName(runner.program, billingWidget)).toBe('Widget')
    expect(optionalTypeName(runner.program, meteringPlan)).toBe('MeteringPlan')
    expect(optionalTypeName(runner.program, plan)).toBe('Plan')
  })
})
