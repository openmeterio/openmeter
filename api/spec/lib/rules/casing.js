import { createRule, paramMessage } from '@typespec/compiler'
import {
  isCamelCaseNoAcronyms,
  isPascalCaseNoAcronyms,
  isSnakeCase,
} from './utils.js'

export const casingRule = createRule({
  name: 'casing',
  severity: 'warning',
  description: 'Ensure proper casing style.',
  messages: {
    name: paramMessage`The names of ${'type'} types must use ${'casing'}`,
    value: paramMessage`The values of ${'type'} types must use ${'casing'}`,
  },
  create: (context) => ({
    model: (model) => {
      if (!isPascalCaseNoAcronyms(model.name)) {
        context.reportDiagnostic({
          format: { type: 'Model', casing: 'PascalCase' },
          target: model,
          messageId: 'name',
        })
      }
    },
    modelProperty: (property) => {
      if (
        (property.name !== '_' && !isCamelCaseNoAcronyms(property.name)) ||
        isPascalCaseNoAcronyms(property.name)
      ) {
        context.reportDiagnostic({
          format: { type: 'Model Property', casing: 'camelCase' },
          target: property,
          messageId: 'name',
        })
      }
    },
  }),
})

export const casingAIPRule = createRule({
  name: 'casing-aip',
  severity: 'warning',
  description: 'Ensure proper casing style for AIP naming conventions.',
  messages: {
    name: paramMessage`The names of ${'type'} types must use ${'casing'}`,
  },
  create: (context) => ({
    model: (model) => {
      if (!isPascalCaseNoAcronyms(model.name)) {
        context.reportDiagnostic({
          format: { type: 'Model', casing: 'PascalCase' },
          target: model,
          messageId: 'name',
        })
      }
    },
    modelProperty: (property) => {
      const isPath = property.decorators.find(
        (d) => d.decorator.name === '$path',
      )

      if (isPath) {
        if (!isCamelCaseNoAcronyms(property.name)) {
          context.reportDiagnostic({
            format: { type: 'Model Property', casing: 'camelCase' },
            target: property,
            messageId: 'name',
          })
        }

        return
      }

      if (
        !['_', 'contentType'].includes(property.name) &&
        !isSnakeCase(property.name)
      ) {
        context.reportDiagnostic({
          format: { type: 'Model Property', casing: 'snake_case' },
          target: property,
          messageId: 'name',
        })
      }
    },
    enum: (model) => {
      // Check enum name is PascalCase
      if (!isPascalCaseNoAcronyms(model.name)) {
        context.reportDiagnostic({
          format: { type: 'Enum', casing: 'PascalCase' },
          target: model,
          messageId: 'name',
        })
      }

      // Check enum member names are PascalCase
      for (const member of model.members.values()) {
        if (!isPascalCaseNoAcronyms(member.name)) {
          context.reportDiagnostic({
            format: { type: 'Enum Member', casing: 'PascalCase' },
            target: member,
            messageId: 'name',
          })
        }
      }
    },
  }),
})

export const casingAIPErrorsRule = createRule({
  name: 'casing-aip-errors',
  severity: 'error',
  description: 'Ensure proper casing style for AIP naming conventions.',
  messages: {
    value: paramMessage`The values of ${'type'} types must use ${'casing'}`,
  },
  create: (context) => ({
    enum: (model) => {
      // Check enum member values are snake_case
      for (const member of model.members.values()) {
        if (member.value && typeof member.value === 'string') {
          if (!isSnakeCase(member.value)) {
            context.reportDiagnostic({
              format: { type: 'Enum Value', casing: 'snake_case' },
              target: member,
              messageId: 'value',
            })
          }
        }
      }
    },
  }),
})
