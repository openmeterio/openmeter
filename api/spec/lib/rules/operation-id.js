import { createRule, paramMessage } from '@typespec/compiler'
import { isKebabCase } from './utils.js'

export const operationIdKebabCaseRule = createRule({
  name: 'operation-id-kebab-case',
  severity: 'error',
  description: 'Ensure @operationId values are in kebab-case.',
  messages: {
    default: paramMessage`The operationId "${'operationId'}" should be in kebab-case.`,
  },
  create: (context) => ({
    operation: (node) => {
      const operationIdDecorator = node.decorators.find(
        (d) => d.decorator.name === '$operationId',
      )

      if (operationIdDecorator) {
        const operationId = operationIdDecorator.args[0]?.jsValue
        if (
          operationId &&
          typeof operationId === 'string' &&
          !isKebabCase(operationId)
        ) {
          context.reportDiagnostic({
            format: {
              operationId,
            },
            target: node,
            messageId: 'default',
          })
        }
      }
    },
  }),
})
