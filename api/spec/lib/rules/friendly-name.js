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
      const hasFriendlyName = node.decorators.some(
        (d) => d.decorator.name === '$friendlyName',
      )

      if (!hasFriendlyName) {
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

export const friendlyNameAIPRule = createRule({
  name: 'friendly-name-aip',
  severity: 'error',
  description: 'Ensure friendlyName decorator.',
  messages: {
    default: paramMessage`The ${'type'} ${'name'} must have a friendlyName decorator.`,
    avoid: paramMessage`The ${'type'} ${'name'} should not have a friendlyName decorator.`,
  },
  create: (context) => ({
    interface: (node) => {
      const hasFriendlyName = node.decorators.some(
        (d) => d.decorator.name === '$friendlyName',
      )
      const isEndpointsOrOperations =
        node.name.endsWith('Endpoints') || node.name.endsWith('Operations')

      if (isEndpointsOrOperations && hasFriendlyName) {
        context.reportDiagnostic({
          format: {
            type: node.kind,
            name: node.name,
          },
          target: node,
          messageId: 'avoid',
        })
        return
      }

      if (!isEndpointsOrOperations && !hasFriendlyName) {
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
