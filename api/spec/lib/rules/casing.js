import { createRule, paramMessage } from '@typespec/compiler'
import { isPascalCaseNoAcronyms, isSnakeCase } from './utils.js'

export const casingRule = createRule({
  name: 'casing',
  severity: 'warning',
  description: 'Ensure proper casing style.',
  messages: {
    name: paramMessage`The names of ${'type'} types must use ${'casing'}`,
    value: paramMessage`The values of ${'type'} types must use ${'casing'}`,
  },
  create: (context) => ({
    model: (model) => {
      if (!isPascalCaseNoAcronyms(model.name)) {
        context.reportDiagnostic({
          format: { type: 'Model', casing: 'PascalCase' },
          target: model,
          messageId: 'name',
        })
      }
    },
    enum: (node) => {
      for (const variant of node.members.values()) {
        if (
          typeof variant.name === 'string' &&
          !isSnakeCase(variant.value || variant.name)
        ) {
          context.reportDiagnostic({
            target: variant,
            format: {
              type: 'enum',
              casing: 'snake_case',
            },
            messageId: 'value',
          })
        }
      }
    },
  }),
})
