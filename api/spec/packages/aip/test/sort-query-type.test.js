import {
  createLinterRuleTester,
  createTestHost,
  createTestRunner,
} from '@typespec/compiler/testing'
import { HttpTestLibrary } from '@typespec/http/testing'
import { beforeEach, describe, it } from 'node:test'
import { sortQueryTypeRule } from '../lib/rules/sort-query-type.js'

const preamble = `
  import "@typespec/http";
  using TypeSpec.Http;

  namespace Common {
    model SortQuery {
      by: string;
      order?: "asc" | "desc";
    }
  }
`

describe('sortQueryTypeRule', () => {
  let ruleTester

  beforeEach(async () => {
    const host = await createTestHost({ libraries: [HttpTestLibrary] })
    const runner = await createTestRunner(host)
    ruleTester = createLinterRuleTester(
      runner,
      sortQueryTypeRule,
      '@openmeter/api-spec-aip',
    )
  })

  it('accepts a sort query parameter using Common.SortQuery', async () => {
    await ruleTester
      .expect(`${preamble}\nop list(@query sort?: Common.SortQuery): void;`)
      .toBeValid()
  })

  it('rejects a scalar sort query parameter', async () => {
    await ruleTester
      .expect(`${preamble}\nop list(@query sort?: string): void;`)
      .toEmitDiagnostics({
        code: '@openmeter/api-spec-aip/sort-query-type',
        message: 'Query parameters named `sort` must use `Common.SortQuery`.',
      })
  })

  it('rejects an alias of Common.SortQuery', async () => {
    await ruleTester
      .expect(
        `
        ${preamble}
        model SortAlias is Common.SortQuery;
        op list(@query sort?: SortAlias): void;
      `,
      )
      .toEmitDiagnostics({
        code: '@openmeter/api-spec-aip/sort-query-type',
        message: 'Query parameters named `sort` must use `Common.SortQuery`.',
      })
  })

  it('checks the effective query parameter name', async () => {
    await ruleTester
      .expect(
        `
        ${preamble}
        op list(@query(#{ name: "sort" }) order?: string): void;
      `,
      )
      .toEmitDiagnostics({
        code: '@openmeter/api-spec-aip/sort-query-type',
        message: 'Query parameters named `sort` must use `Common.SortQuery`.',
      })
  })

  it('ignores unrelated query parameters', async () => {
    await ruleTester
      .expect(`${preamble}\nop list(@query sorting?: string): void;`)
      .toBeValid()
  })
})
