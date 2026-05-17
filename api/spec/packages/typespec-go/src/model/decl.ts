/**
 * Declarations the emitter produces.
 *
 * Each Decl becomes a single Go type definition (struct, type alias, enum, ...).
 */
import type { GoType } from "./type.js";

export type Decl = StructDecl | EnumDecl | DiscriminatedUnionDecl | HeuristicUnionDecl | AliasDecl;

/** A struct declaration. */
export interface StructDecl {
  readonly kind: "struct";
  readonly name: string;
  readonly doc?: string;
  readonly fields: readonly StructField[];
  /**
   * If set, this struct is a variant of a discriminated union. Its `type` field
   * will be a singleton enum (e.g. `TypeFlatFee` with the single value `"flat_fee"`).
   */
  readonly discriminatorVariant?: DiscriminatorVariantInfo;
}

export interface DiscriminatorVariantInfo {
  /** Parent union name (e.g. "BillingCharge"). */
  readonly unionName: string;
  /** This variant's discriminator value (e.g. "flat_fee"). */
  readonly value: string;
  /** Name of the singleton enum type (e.g. "TypeFlatFee"). */
  readonly singletonEnumName: string;
}

export interface StructField {
  /** Go field name (PascalCase). */
  readonly name: string;
  /** JSON property name (snake_case, from the spec). */
  readonly jsonName: string;
  /** Field type. */
  readonly type: GoType;
  /** Field is optional — emitted as `*T` unless type is already a slice/map. */
  readonly optional: boolean;
  /** Documentation string for the field. */
  readonly doc?: string;
}

/** A string-valued enum. */
export interface EnumDecl {
  readonly kind: "enum";
  readonly name: string;
  readonly doc?: string;
  readonly members: readonly EnumMember[];
  /**
   * When true, an `<Name>Unknown` member is appended for forward compatibility
   * (matches Speakeasy's forwardCompatibleEnumsByDefault behavior).
   *
   * v1: always true for top-level enums, false for singleton variant enums.
   */
  readonly forwardCompatible: boolean;
}

export interface EnumMember {
  /** Go const name suffix (e.g. for enum `MeterAggregation`, member `Sum` → `MeterAggregationSum`). */
  readonly name: string;
  /** JSON value (e.g. `"sum"`). */
  readonly value: string;
}

/**
 * A discriminated union (e.g. `BillingCharge` with `type` discriminator).
 *
 * Emits:
 *   type <Name>Type string + const block (parent enum, includes UNKNOWN)
 *   type <Name> struct {
 *     <Variant1> *<Variant1Type> `union:"member"`
 *     ...
 *     UnknownRaw json.RawMessage `union:"unknown"`
 *     Type <Name>Type
 *   }
 *   func Create<Name><Variant>(...) <Name> { ... }
 *   func (u <Name>) MarshalJSON() / UnmarshalJSON(...)
 *
 * Each variant struct (declared separately as a StructDecl with discriminatorVariant set)
 * carries its own singleton `Type<Variant>` enum field.
 */
export interface DiscriminatedUnionDecl {
  readonly kind: "discriminated-union";
  readonly name: string;
  readonly doc?: string;
  /** Name of the JSON property carrying the discriminator (e.g. "type"). */
  readonly discriminatorProperty: string;
  /**
   * If set, the discriminator enum is a separately-declared TypeSpec enum
   * (e.g. `AppType`). The emitter will reference it instead of synthesizing
   * a parallel `<Name>Type` enum and const block.
   *
   * If undefined, the emitter synthesizes a `<Name>Type` enum and const block.
   */
  readonly discriminatorEnumName?: string;
  readonly variants: readonly DiscriminatedVariant[];
}

export interface DiscriminatedVariant {
  /** Variant field name on the union struct (e.g. "BillingFlatFeeCharge"). */
  readonly name: string;
  /** Discriminator value for this variant (e.g. "flat_fee"). */
  readonly value: string;
  /** Reference to the variant's struct (a named Go type). */
  readonly typeRef: { readonly name: string; readonly pkg: string };
}

/**
 * A non-discriminated union (e.g. `ULIDFieldFilterUnion`, filter unions).
 *
 * Emits the same shape as DiscriminatedUnion but uses heuristic unmarshal
 * via `utils.PickBestUnionCandidate`.
 */
export interface HeuristicUnionDecl {
  readonly kind: "heuristic-union";
  readonly name: string;
  readonly doc?: string;
  readonly variants: readonly HeuristicVariant[];
}

export interface HeuristicVariant {
  /** Variant field name on the union struct. */
  readonly name: string;
  /** Variant Go type. */
  readonly type: GoType;
}

/** A scalar alias (e.g. `type ULID = string`). */
export interface AliasDecl {
  readonly kind: "alias";
  readonly name: string;
  readonly doc?: string;
  readonly target: GoType;
}
