import { createRule, paramMessage } from '@typespec/compiler'
import { isPascalCaseNoAcronyms } from './utils.js'

export const discriminatedUnionsRule = createRule({
  name: 'discriminated-unions',
  severity: 'error',
  description:
    'Ensure discriminated unions on model properties does not have any additional decorators or description.',
  messages: {
    decorators: paramMessage`Discriminated union properties must not have additional decorators: ${'decorators'}`,
  },
  create: (context) => ({
    model: (model) => {
      for (const [name, field] of model.properties) {
        if (
          field.type.kind === 'Union' &&
          field.type.decorators.some(
            (d) => d.decorator.name === '$discriminator',
          ) &&
          field.decorators.filter(
            (d) => !['$body', '$visibility'].includes(d.decorator.name),
          ).length > 0
        ) {
          context.reportDiagnostic({
            format: {
              decorators: field.decorators
                .map((d) => d.decorator.name)
                .join(', '),
            },
            target: field,
            messageId: 'decorators',
          })
        }
      }
    },
  }),
})
