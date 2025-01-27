import fs from 'fs'
import openapiTS, { astToString } from 'openapi-typescript'
import { factory, SyntaxKind } from 'typescript'

const DATE = factory.createTypeReferenceNode(factory.createIdentifier('Date')) // `Date`
const NULL = factory.createLiteralTypeNode(factory.createNull()) // `null`
const STRING = factory.createKeywordTypeNode(SyntaxKind.StringKeyword) // `string`

const schema = new URL('../../../openapi.cloud.yaml', import.meta.url)

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

fs.writeFileSync('./src/client/schemas.d.ts', contents)
