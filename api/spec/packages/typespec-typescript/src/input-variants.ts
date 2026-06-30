import { type Model, type Program, type Type } from '@typespec/compiler'
import { bodyProperties } from './utils.jsx'

/**
 * The input shape of a model differs from its output shape only when a defaulted
 * property — anywhere in its reachable subtree — flips from required (output) to
 * optional (input). This computes the set of models for which an `…Input`
 * variant interface must be emitted.
 *
 * A model diverges-on-input iff it has a defaulted property, or any property
 * type (descending through arrays, records, unions, tuples, and `baseModel`,
 * matching the schema walker) transitively reaches a model that diverges.
 */
export function computeDivergentModels(
  program: Program,
  models: Model[],
): Set<Model> {
  const diverges = new Set<Model>()
  const memo = new Map<Model, boolean>()
  const stack = new Set<Model>()

  const reaches = (type: Type): boolean => {
    switch (type.kind) {
      case 'Model':
        return modelDiverges(type)
      case 'ModelProperty':
        return reaches(type.type)
      case 'Union':
        for (const variant of type.variants.values()) {
          if (reaches(variant.type)) {
            return true
          }
        }
        return false
      case 'Tuple':
        return type.values.some(reaches)
      default:
        return false
    }
  }

  const modelDiverges = (model: Model): boolean => {
    const cached = memo.get(model)
    if (cached !== undefined) {
      return cached
    }
    // A cycle that reaches back to an in-progress model contributes nothing on
    // its own; the originating frame still sees its other branches.
    if (stack.has(model)) {
      return false
    }
    stack.add(model)

    let result = false
    if (model.indexer && reaches(model.indexer.value)) {
      result = true
    }
    if (!result && model.baseModel && modelDiverges(model.baseModel)) {
      result = true
    }
    if (!result) {
      for (const prop of bodyProperties(program, model)) {
        if (prop.defaultValue !== undefined || reaches(prop.type)) {
          result = true
          break
        }
      }
    }

    stack.delete(model)
    memo.set(model, result)
    if (result) {
      diverges.add(model)
    }
    return result
  }

  for (const model of models) {
    modelDiverges(model)
  }
  return diverges
}

/** The interface name of a model's input variant. */
export function inputVariantName(outputName: string): string {
  return `${outputName}Input`
}
