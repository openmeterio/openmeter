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
      if (
        target.name &&
        !target.decorators.find(
          (d) => d.decorator?.name === 'docFromCommentDecorator',
        )
      ) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }

      for (const [name, property] of target.properties) {
        if (
          target.name &&
          name &&
          !property.decorators.find(
            (d) => d.decorator?.name === 'docFromCommentDecorator',
          )
        ) {
          context.reportDiagnostic({
            target: property,
            format: {
              name: `${target.name}.${name}`,
            },
          })
        }
      }
    },
    enum: (target) => {
      if (
        target.name &&
        !target.decorators.find(
          (d) => d.decorator?.name === 'docFromCommentDecorator',
        )
      ) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
    union: (target) => {
      if (
        target.name &&
        !target.decorators.find(
          (d) => d.decorator?.name === 'docFromCommentDecorator',
        )
      ) {
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
