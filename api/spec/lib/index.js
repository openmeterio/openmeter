import { defineLinter } from '@typespec/compiler'
import { casingRule } from './rules/casing.js'
import { docDecoratorRule } from './rules/docs.js'
import { friendlyNameRule } from './rules/friendlyName.js'
import { discriminatedUnionsRule } from './rules/discriminatedUnions.js'
import { operationSummaryRule } from './rules/operationSummary.js'

const rules = [
  casingRule,
  docDecoratorRule,
  friendlyNameRule,
  discriminatedUnionsRule,
  operationSummaryRule,
]

// Linter experimentation
// See: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
export const $linter = defineLinter({
  rules,
})
