import { createRule, paramMessage } from '@typespec/compiler'
import { isCamelCaseNoAcronyms, isPascalCaseNoAcronyms } from './utils.js'

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
    modelProperty: (property) => {
      if (
        !isCamelCaseNoAcronyms(property.name) ||
        isPascalCaseNoAcronyms(property.name)
      ) {
        context.reportDiagnostic({
          format: { type: 'Model Property', casing: 'camelCase' },
          target: property,
          messageId: 'name',
        })
      }
    },
  }),
})
