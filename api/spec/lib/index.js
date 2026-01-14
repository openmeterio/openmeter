import { defineLinter } from '@typespec/compiler'
import {
  casingAIPErrorsRule,
  casingAIPRule,
  casingRule,
} from './rules/casing.js'
import { docDecoratorRule } from './rules/docs.js'
import { friendlyNameAIPRule, friendlyNameRule } from './rules/friendly-name.js'
import { operationSummaryRule } from './rules/operation-summary.js'
import { operationIdKebabCaseRule } from './rules/operation-id.js'
import { noNullableRule } from './rules/no-nullable.js'
import { compositionOverInheritanceRule } from './rules/composition-over-inheritance.js'
import { repeatedPrefixGroupingRule } from './rules/field-prefix.js'

const packageName = '@openmeter/api-spec'

// See example rules: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
const rules = [
  casingRule,
  casingAIPRule,
  casingAIPErrorsRule,
  docDecoratorRule,
  friendlyNameRule,
  friendlyNameAIPRule,
  noNullableRule,
  operationSummaryRule,
  operationIdKebabCaseRule,
  compositionOverInheritanceRule,
  repeatedPrefixGroupingRule,
]

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
        [`${packageName}/${casingAIPErrorsRule.name}`]: true,
        [`${packageName}/${docDecoratorRule.name}`]: true,
        [`${packageName}/${friendlyNameAIPRule.name}`]: true,
        [`${packageName}/${noNullableRule.name}`]: true,
        [`${packageName}/${operationSummaryRule.name}`]: true,
        [`${packageName}/${operationIdKebabCaseRule.name}`]: true,
        [`${packageName}/${compositionOverInheritanceRule.name}`]: true,
        [`${packageName}/${repeatedPrefixGroupingRule.name}`]: true,
      },
    },
  },
})
