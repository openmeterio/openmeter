import { createRule, getNamespaceFullName } from '@typespec/compiler'
import { getQueryParamName } from '@typespec/http'

export const sortQueryTypeRule = createRule({
  name: 'sort-query-type',
  severity: 'warning',
  description:
    'Require query parameters named `sort` to use `Common.SortQuery`.',
  messages: {
    default: 'Query parameters named `sort` must use `Common.SortQuery`.',
  },
  create(context) {
    return {
      modelProperty: (property) => {
        if (getQueryParamName(context.program, property) !== 'sort') {
          return
        }

        const type = property.type
        if (
          type.kind === 'Model' &&
          type.name === 'SortQuery' &&
          type.namespace &&
          getNamespaceFullName(type.namespace) === 'Common'
        ) {
          return
        }

        context.reportDiagnostic({ target: property })
      },
    }
  },
})
