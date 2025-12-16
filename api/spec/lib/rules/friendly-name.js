import { createRule, paramMessage } from '@typespec/compiler'

export const friendlyNameRule = createRule({
  name: 'friendlyName',
  severity: 'error',
  description: 'Ensure friendlyName decorator.',
  messages: {
    default: paramMessage`The ${'type'} ${'name'} must have a friendlyName decorator.`,
  },
  create: (context) => ({
    interface: (node) => {
      if (
        node.name &&
        !node.name.endsWith('Operations') &&
        !node.name.endsWith('Endpoints') &&
        !node.decorators.some((d) => d.decorator.name === '$friendlyName')
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
    model: (node) => {
      if (
        node.name &&
        !node.decorators.some((d) => d.decorator.name === '$friendlyName')
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
    enum: (node) => {
      if (
        node.name &&
        !node.decorators.some((d) => d.decorator.name === '$friendlyName')
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
    union: (node) => {
      if (
        node.name &&
        !node.decorators.some((d) => d.decorator.name === '$friendlyName')
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
