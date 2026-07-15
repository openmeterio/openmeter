import { createTypeSpecLibrary, ignoreDiagnostics } from '@typespec/compiler'

const queryCodecKey = Symbol.for('@openmeter/typespec-sdk.query-codec')

export const $lib = createTypeSpecLibrary({
  name: '@openmeter/typespec-sdk',
  diagnostics: {
    'invalid-query-codec-wire-type': {
      severity: 'error',
      messages: {
        default: 'query codec sort must encode to a string-compatible scalar',
      },
    },
  },
})

const { reportDiagnostic } = $lib

export function $queryCodec(context, target, codec, wireType) {
  const stringType = context.program.checker.getStdType('string')
  const compatible = ignoreDiagnostics(
    context.program.checker.isTypeAssignableTo(
      wireType,
      stringType,
      context.decoratorTarget,
    ),
  )
  if (!compatible) {
    reportDiagnostic(context.program, {
      code: 'invalid-query-codec-wire-type',
      target: context.decoratorTarget,
    })
    return
  }

  context.program.stateMap(queryCodecKey).set(target, { codec, wireType })
}

export const $decorators = {
  'OpenMeter.Sdk': {
    queryCodec: $queryCodec,
  },
}

export function getQueryCodec(program, target) {
  const codecs = program.stateMap(queryCodecKey)
  for (let property = target; property; property = property.sourceProperty) {
    const codec = codecs.get(property)
    if (codec) {
      return codec
    }
  }
  return undefined
}
