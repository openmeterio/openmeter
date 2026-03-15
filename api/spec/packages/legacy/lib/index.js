import { defineLinter } from '@typespec/compiler'
import { casingRule } from './rules/casing.js'
import { docDecoratorRule } from './rules/docs.js'
import { friendlyNameRule } from './rules/friendly-name.js'
import { operationSummaryRule } from './rules/operation-summary.js'
import { compositionOverInheritanceRule } from './rules/composition-over-inheritance.js'

// See example rules: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
const rules = [
  casingRule,
  docDecoratorRule,
  friendlyNameRule,
  operationSummaryRule,
  compositionOverInheritanceRule,
]

export const $linter = defineLinter({
  rules,
})
