import { createRule, paramMessage, getDoc } from '@typespec/compiler'

export const docDecoratorRule = createRule({
  name: 'doc-decorator',
  severity: 'warning',
  description: 'Ensure documentation.',
  messages: {
    default: paramMessage`Missing documentation for ${'name'} ${'type'}`,
  },
  create: (context) => ({
    model: (target) => {
      if (target.name && !getDoc(context.program, target)) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }

      if (target.name.endsWith('Response')) {
        return
      }

      for (const [name, property] of target.properties) {
        if (
          target.name &&
          name &&
          !['_', 'contentType'].includes(name) &&
          !getDoc(context.program, property)
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
      if (target.name && !getDoc(context.program, target)) {
        context.reportDiagnostic({
          target,
          format: {
            name: target.name,
          },
        })
      }
    },
    union: (target) => {
      if (target.name && !getDoc(context.program, target)) {
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
