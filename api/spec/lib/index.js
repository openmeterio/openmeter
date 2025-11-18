import { defineLinter } from '@typespec/compiler'
import { casingAIPRule, casingRule } from './rules/casing.js'
import { docDecoratorRule } from './rules/docs.js'
import { friendlyNameRule } from './rules/friendlyName.js'
import { operationSummaryRule } from './rules/operationSummary.js'
import { noNullableRule } from './rules/no-nullable.js'
import { compositionOverInheritanceRule } from './rules/composition-over-inheritance.js'

const packageName = '@openmeter/api-spec'

const rules = [
  casingRule,
  casingAIPRule,
  docDecoratorRule,
  friendlyNameRule,
  noNullableRule,
  operationSummaryRule,
  compositionOverInheritanceRule,
]

// Linter experimentation
// See: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
export const $linter = defineLinter({
  rules,
  ruleSets: {
    legacy: {
      enable: {
        [`${packageName}/${casingRule.name}`]: true,
        [`${packageName}/${docDecoratorRule.name}`]: true,
        [`${packageName}/${friendlyNameRule.name}`]: true,
        [`${packageName}/${noNullableRule.name}`]: true,
        [`${packageName}/${operationSummaryRule.name}`]: true,
        [`${packageName}/${compositionOverInheritanceRule.name}`]: true,
      },
    },
    aip: {
      enable: {
        [`${packageName}/${casingAIPRule.name}`]: true,
        [`${packageName}/${docDecoratorRule.name}`]: true,
        [`${packageName}/${friendlyNameRule.name}`]: true,
        [`${packageName}/${noNullableRule.name}`]: true,
        [`${packageName}/${operationSummaryRule.name}`]: true,
        [`${packageName}/${compositionOverInheritanceRule.name}`]: true,
      },
    },
  },
})
