import { createRule, paramMessage } from '@typespec/compiler'

export const operationSummaryRule = createRule({
  name: 'operation-summary',
  severity: 'warning',
  description: 'Ensure operation summary.',
  messages: {
    default: paramMessage`The ${'type'} ${'name'} must have a summary decorator.`,
  },
  create: (context) => ({
    operation: (node) => {
      if (
        node.name &&
        !node.decorators.some((d) => d.decorator.name === '$summary')
      ) {
        context.reportDiagnostic({
          format: {
            type: node.kind,
            name: node.name,
          },
          target: node,
          messageId: 'default',
        })
      }
    },
  }),
})
