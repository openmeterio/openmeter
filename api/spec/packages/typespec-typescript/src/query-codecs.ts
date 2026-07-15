import type { Program, Scalar } from '@typespec/compiler'

export type QueryCodecName = 'sort'

export interface QueryCodec {
  name: QueryCodecName
  wireType: Scalar
}

/**
 * Returns the SDK codec reserved for an HTTP query parameter name.
 *
 * The AIP `sort-query-type` rule guarantees that `sort` uses
 * `Common.SortQuery`; this mapper owns only the transport representation.
 */
export function queryCodecForParameter(
  program: Program,
  parameterName: string,
): QueryCodec | undefined {
  switch (parameterName) {
    case 'sort':
      return {
        name: 'sort',
        wireType: program.checker.getStdType('string'),
      }
    default:
      return undefined
  }
}
