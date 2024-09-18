import { createRule, paramMessage } from '@typespec/compiler'

export const docDecoratorRule = createRule({
  name: 'doc-decorator',
  severity: 'warning',
  description: 'Ensure documentation.',
  messages: {
    default: paramMessage`Missing documentation for ${'name'} ${'type'}`,
  },
  create: (context) => ({
    model: (target) => {
      if (target.name && !target.decorators.find((d) => d.decorator?.name === '$docFromComment')) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
    enum: (target) => {
      if (target.name && !target.decorators.find((d) => d.decorator?.name === '$docFromComment')) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
    union: (target) => {
      if (target.name && !target.decorators.find((d) => d.decorator?.name === '$docFromComment')) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
  }),
})
