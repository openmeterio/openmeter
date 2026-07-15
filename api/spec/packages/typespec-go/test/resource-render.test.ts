import { createTestHost, createTestRunner } from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { OpenAPITestLibrary } from '@typespec/openapi/testing'
import { TypeSpecSdkTestLibrary } from '@openmeter/typespec-sdk/testing'
import { describe, expect, it } from 'vitest'
import type { Program } from '@typespec/compiler'
import {
  collectHttpOperations,
  describeOperations,
  type GoOperation,
} from '../dist/operations.js'
import {
  deepObjectName,
  isPlainPageEnvelope,
  localName,
  queryScalarValue,
  resolveListParamsNames,
} from '../dist/components/GoResource.js'

async function compileResource(
  code: string,
): Promise<{ program: Program; operations: GoOperation[] }> {
  const host = await createTestHost({
    libraries: [HttpTestLibrary, OpenAPITestLibrary, TypeSpecSdkTestLibrary],
  })
  const runner = await createTestRunner(host)
  await runner.compile(`
    import "@typespec/http";
    import "@typespec/openapi";
    import "@openmeter/typespec-sdk";
    using TypeSpec.Http;
    using TypeSpec.OpenAPI;
    ${code}
  `)

  const operations = collectHttpOperations(runner.program)
  return {
    program: runner.program,
    operations: describeOperations(runner.program, 'Test', operations),
  }
}

function operationNamed(operations: GoOperation[], name: string): GoOperation {
  const operation = operations.find(
    (candidate) => candidate.operation.name === name,
  )
  if (!operation) {
    throw new Error(`missing operation ${name} in fixture`)
  }
  return operation
}

const pageFixturePreamble = `
  @service namespace Test;

  model SortQuery {
    by: string;
    order?: "asc" | "desc";
  }

  model Item {
    id: string;
  }

  model ItemPageMeta {
    total: int32;
  }

  model ItemPage {
    data: Item[];
    meta: ItemPageMeta;
  }
`

describe('list params struct naming', () => {
  it('splits shared-element params structs when query shapes differ', async () => {
    const { program, operations } = await compileResource(`
      ${pageFixturePreamble}

      @route("/items")
      interface Operations {
        @get
        @operationId("list-items")
        op listItems(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
          @OpenMeter.Sdk.queryCodec("sort", string)
          @query sort?: SortQuery,
        ): ItemPage;

        @get
        @route("/archived")
        @operationId("list-archived-items")
        op listArchived(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): ItemPage;
      }
    `)

    const names = resolveListParamsNames(program, operations)
    expect(names.get(operationNamed(operations, 'listItems'))).toBe(
      'ListItemsParams',
    )
    expect(names.get(operationNamed(operations, 'listArchived'))).toBe(
      'ListArchivedItemsParams',
    )

    // The outcome must not depend on which operation is discovered first.
    const reversed = resolveListParamsNames(program, [...operations].reverse())
    expect(reversed.get(operationNamed(operations, 'listItems'))).toBe(
      'ListItemsParams',
    )
    expect(reversed.get(operationNamed(operations, 'listArchived'))).toBe(
      'ListArchivedItemsParams',
    )
  })

  it('shares one element-named params struct when query shapes match', async () => {
    const { program, operations } = await compileResource(`
      ${pageFixturePreamble}

      @route("/items")
      interface Operations {
        @get
        @operationId("list-items")
        op listItems(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): ItemPage;

        @get
        @route("/archived")
        @operationId("list-archived-items")
        op listArchived(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): ItemPage;
      }
    `)

    const names = resolveListParamsNames(program, operations)
    expect(names.get(operationNamed(operations, 'listItems'))).toBe(
      'ItemListParams',
    )
    expect(names.get(operationNamed(operations, 'listArchived'))).toBe(
      'ItemListParams',
    )
  })
})

describe('All iterator emission rule', () => {
  it('accepts only the plain {data, meta} page envelope', async () => {
    const { program, operations } = await compileResource(`
      ${pageFixturePreamble}

      model QueryPage {
        data: Item[];
        meta: ItemPageMeta;
        errors: string[];
      }

      model StatusItemPage {
        @statusCode _: 200;
        data: Item[];
        meta: ItemPageMeta;
      }

      @route("/items")
      interface Operations {
        @get
        @operationId("list-plain")
        op listPlain(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): ItemPage;

        @get
        @route("/query")
        @operationId("list-partial")
        op listPartial(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): QueryPage;

        @get
        @route("/status")
        @operationId("list-status")
        op listStatus(
          @query(#{ style: "deepObject", explode: true })
          page?: {
            size?: integer;
            number?: integer;
          },
        ): StatusItemPage;
      }
    `)

    const plain = operationNamed(operations, 'listPlain')
    expect(plain.pagination).toBe('page')
    expect(isPlainPageEnvelope(program, plain.response)).toBe(true)

    // A paginated response carrying extra fields (partial-failure errors) must
    // not get an All iterator that would silently drop them.
    const partial = operationNamed(operations, 'listPartial')
    expect(partial.pagination).toBe('page')
    expect(isPlainPageEnvelope(program, partial.response)).toBe(false)

    // HTTP metadata properties (status code, headers) do not count as payload.
    const status = operationNamed(operations, 'listStatus')
    expect(isPlainPageEnvelope(program, status.response)).toBe(true)

    expect(isPlainPageEnvelope(program, undefined)).toBe(false)
  })
})

describe('path parameter local names', () => {
  it('lowercases whole-acronym parameters', () => {
    expect(localName('id')).toBe('id')
    expect(localName('ulid')).toBe('ulid')
  })

  it('keeps mixed-word casing intact', () => {
    expect(localName('customerId')).toBe('customerID')
    expect(localName('priceId')).toBe('priceID')
    expect(localName('llmModel')).toBe('llmModel')
  })

  it('renames parameters colliding with generated method locals', () => {
    expect(localName('request')).toBe('requestParam')
    expect(localName('path')).toBe('pathParam')
    expect(localName('params')).toBe('paramsParam')
    expect(localName('page')).toBe('pageParam')
    expect(localName('s')).toBe('sParam')
  })
})

describe('query rendering helpers', () => {
  it('strips the Params suffix from deep-object type names', async () => {
    const { operations } = await compileResource(`
      @service namespace Test;

      model ItemFilter {
        key?: string;
      }

      @route("/items")
      interface Operations {
        @get
        op list(
          @query(#{ style: "deepObject", explode: true }) filter?: ItemFilter,
          @query(#{ style: "deepObject", explode: true }) options?: ItemFilter,
        ): string[];
      }
    `)

    const list = operationNamed(operations, 'list')
    const filter = list.queryParams.find(
      (parameter) => parameter.name === 'filter',
    )!
    const options = list.queryParams.find(
      (parameter) => parameter.name === 'options',
    )!

    expect(deepObjectName('ItemListParams', filter)).toBe('ItemFilter')
    expect(deepObjectName('GetCustomerCreditBalanceParams', filter)).toBe(
      'GetCustomerCreditBalanceFilter',
    )
    expect(deepObjectName('ItemListParams', options)).toBe('ItemListOptions')
  })

  it('omits no-op query scalar conversions', async () => {
    const { program, operations } = await compileResource(`
      @service namespace Test;

      enum Color {
        red,
        green,
      }

      @route("/items")
      interface Operations {
        @get
        op list(
          @query code?: string,
          @query color?: Color,
        ): string[];
      }
    `)

    const list = operationNamed(operations, 'list')
    const code = list.queryParams.find(
      (parameter) => parameter.name === 'code',
    )!
    const color = list.queryParams.find(
      (parameter) => parameter.name === 'color',
    )!

    // A Go string needs no conversion (and never doubled parentheses).
    expect(queryScalarValue(program, code.type, '*p.Code')).toBe('*p.Code')
    expect(queryScalarValue(program, code.type, 'value')).toBe('value')
    // Named string-like types still convert, with single parentheses.
    expect(queryScalarValue(program, color.type, '*p.Color')).toBe(
      'string(*p.Color)',
    )
  })
})
