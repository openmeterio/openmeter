import { defineLinter } from '@typespec/compiler'
import { casingRule } from './rules/casing.js'
import { docDecoratorRule } from './rules/docs.js'
import { friendlyNameRule } from './rules/friendlyName.js'

const rules = [casingRule, docDecoratorRule, friendlyNameRule]

// Linter experimentation
// See: https://github.com/Azure/typespec-azure/tree/main/packages/typespec-azure-core/src/rules
export const $linter = defineLinter({
  rules,
})
