import { createRule, paramMessage } from '@typespec/compiler'

export const repeatedPrefixGroupingRule = createRule({
  name: 'repeated-prefix-grouping',
  severity: 'warning',
  description: 'Disallow repeated <prefix>_* field prefixes within a model.',
  messages: {
    default: paramMessage`Repeated "${'prefix'}_" field prefix detected (${`fields`}). Group them under a "${'prefix'}" object.`,
  },
  create(context) {
    return {
      model: (model) => {
        if (!model?.properties || model.properties.size === 0) {
          return
        }

        const byPrefix = new Map()

        for (const prop of model.properties.values()) {
          const name = prop?.name
          if (!name) {
            continue
          }

          // "prefix" is everything before the first underscore.
          const underscoreIndex = name.indexOf('_')
          if (underscoreIndex <= 0) {
            continue
          }

          const prefix = name.slice(0, underscoreIndex)
          if (!prefix) continue

          const list = byPrefix.get(prefix) ?? []
          list.push(name)
          byPrefix.set(prefix, list)
        }

        for (const [prefix, fields] of byPrefix.entries()) {
          if (fields.length <= 1) {
            continue
          }

          context.reportDiagnostic({
            target: model,
            messageId: 'default',
            format: {
              prefix,
              fields: fields.sort().join(', '),
            },
          })
        }
      },
    }
  },
})
