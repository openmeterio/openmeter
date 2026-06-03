import fs from 'node:fs'
import openapiTS, { astToString } from 'openapi-typescript'
import { factory, SyntaxKind } from 'typescript'

// v3 compatibility shim — types generator. Mirrors scripts/generate.ts but
// points at the v3 OpenAPI spec (api/v3/openapi.yaml) and writes src/v3/schemas.ts.
// See V3_SHIM_PLAN.md. The v1 `#/components/schemas/Event` optional-field hack is
// intentionally omitted: the v3 MeteringEvent schema already expresses optionality
// correctly, so only the date-time transform is needed.

const DATE = factory.createTypeReferenceNode(factory.createIdentifier('Date')) // `Date`
const NULL = factory.createLiteralTypeNode(factory.createNull()) // `null`
const STRING = factory.createKeywordTypeNode(SyntaxKind.StringKeyword) // `string`

const schema = new URL('../../../v3/openapi.yaml', import.meta.url)

const ast = await openapiTS(schema, {
  defaultNonNullable: false,
  rootTypes: true,
  rootTypesNoSchemaPrefix: true,
  transform(schemaObject, metadata) {
    if (schemaObject.format === 'date-time') {
      const allowString =
        (metadata.schema &&
          'in' in metadata.schema &&
          metadata.schema.in === 'query') ||
        metadata.path?.includes('/parameters/query')

      // allow string in query parameters
      if (allowString) {
        return schemaObject.nullable
          ? factory.createUnionTypeNode([DATE, NULL, STRING])
          : factory.createUnionTypeNode([DATE, STRING])
      }

      return schemaObject.nullable
        ? factory.createUnionTypeNode([DATE, NULL])
        : DATE
    }
  },
})

const contents = astToString(ast)

fs.mkdirSync('./src/v3', { recursive: true })
fs.writeFileSync('./src/v3/schemas.ts', contents)
