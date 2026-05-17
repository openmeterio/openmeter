/**
 * Go type system used by the emitter.
 *
 * Kept intentionally narrow: only the cases that appear in OpenMeter's spec.
 */

export type GoType =
  | GoScalar
  | GoSlice
  | GoMap
  | GoPointer
  | GoNamed
  | GoTime
  | GoAny;

/** A Go primitive scalar. */
export interface GoScalar {
  readonly kind: "scalar";
  /** `string`, `bool`, `int32`, `int64`, `float32`, `float64`, `[]byte` */
  readonly name: string;
}

export interface GoSlice {
  readonly kind: "slice";
  readonly element: GoType;
}

export interface GoMap {
  readonly kind: "map";
  /** Always `string` in OpenMeter's spec; kept generic for completeness. */
  readonly key: GoType;
  readonly value: GoType;
}

export interface GoPointer {
  readonly kind: "pointer";
  readonly element: GoType;
}

/** Reference to a named declaration in the generated SDK (struct/enum/union/alias). */
export interface GoNamed {
  readonly kind: "named";
  readonly name: string;
  /** Package the type lives in, relative to the module root (e.g. "models/components"). */
  readonly pkg: string;
}

/** `time.Time`. */
export interface GoTime {
  readonly kind: "time";
}

/** `any` (Go `interface{}`). Used when TypeSpec uses `unknown`. */
export interface GoAny {
  readonly kind: "any";
}

export const goString: GoScalar = { kind: "scalar", name: "string" };
export const goBool: GoScalar = { kind: "scalar", name: "bool" };
export const goInt32: GoScalar = { kind: "scalar", name: "int32" };
export const goInt64: GoScalar = { kind: "scalar", name: "int64" };
export const goFloat32: GoScalar = { kind: "scalar", name: "float32" };
export const goFloat64: GoScalar = { kind: "scalar", name: "float64" };
export const goBytes: GoScalar = { kind: "scalar", name: "[]byte" };
export const goTime: GoTime = { kind: "time" };
export const goAny: GoAny = { kind: "any" };

export function ptr(t: GoType): GoPointer {
  return { kind: "pointer", element: t };
}

export function slice(t: GoType): GoSlice {
  return { kind: "slice", element: t };
}

export function mapOf(value: GoType): GoMap {
  return { kind: "map", key: goString, value };
}

export function named(name: string, pkg: string): GoNamed {
  return { kind: "named", name, pkg };
}
