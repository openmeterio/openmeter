import { defineLinter } from '@typespec/compiler'
import { casingErrorsRule, casingRule } from './rules/casing.js'
import { docDecoratorRule, docFormatRule } from './rules/docs.js'
import { friendlyNameRule } from './rules/friendly-name.js'
import { operationSummaryRule } from './rules/operation-summary.js'
import { operationIdKebabCaseRule } from './rules/operation-id.js'
import { noNullableRule } from './rules/no-nullable.js'
import { compositionOverInheritanceRule } from './rules/composition-over-inheritance.js'
import { repeatedPrefixGroupingRule } from './rules/field-prefix.js'

// See example rules: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
const rules = [
  casingRule,
  casingErrorsRule,
  docDecoratorRule,
  docFormatRule,
  friendlyNameRule,
  noNullableRule,
  operationSummaryRule,
  operationIdKebabCaseRule,
  compositionOverInheritanceRule,
  repeatedPrefixGroupingRule,
]

export const $linter = defineLinter({
  rules,
})
