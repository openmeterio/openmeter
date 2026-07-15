import type {
  DecoratorContext,
  ModelProperty,
  Program,
  Scalar,
} from '@typespec/compiler'

export type QueryCodecName = 'sort'

export interface QueryCodec {
  codec: QueryCodecName
  wireType: Scalar
}

export declare function $queryCodec(
  context: DecoratorContext,
  target: ModelProperty,
  codec: QueryCodecName,
  wireType: Scalar,
): void

export declare const $decorators: {
  'OpenMeter.Sdk': {
    queryCodec: typeof $queryCodec
  }
}

export declare function getQueryCodec(
  program: Program,
  target: ModelProperty,
): QueryCodec | undefined
