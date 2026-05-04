import {
  type Children,
  type ComponentContext,
  type ComponentDefinition,
  createContext,
  useContext,
} from "@alloy-js/core";
import type { ObjectPropertyProps, VarDeclarationProps } from "@alloy-js/typescript";
import type {
  Enum,
  EnumMember,
  Model,
  ModelProperty,
  Program,
  Scalar,
  Type,
  Union,
  UnionVariant,
} from "@typespec/compiler";
import { $ } from "@typespec/compiler/typekit";
import { isBuiltIn } from "../utils.jsx";

const getEmitOptionsForTypeSym: unique symbol = Symbol.for("typespec-zod:getEmitOptionsForType");

const getEmitOptionsForTypeKindSym: unique symbol = Symbol.for("typespec-zod:getEmitOptionsForTypeKind");

export type ZodCustomEmitOptions = ZodCustomEmitOptionsClass;
export const ZodCustomEmitOptions = (() => new ZodCustomEmitOptionsClass()) as {
  new (): ZodCustomEmitOptionsClass;
  (): ZodCustomEmitOptionsClass;
};

export class ZodCustomEmitOptionsClass {
  #typeEmitOptions: Map<Type, ZodCustomEmitOptionsBase<any>> = new Map();
  #typeKindEmitOptions: Map<Type["kind"], ZodCustomEmitOptionsBase<any>> = new Map();

  forType<const T extends Type>(type: T, options: ZodCustomEmitOptionsBase<T>) {
    this.#typeEmitOptions.set(type, options);

    return this;
  }

  forTypeKind<const TKind extends Type["kind"]>(
    typeKind: TKind,
    options: ZodCustomEmitOptionsBase<Extract<Type, { kind: TKind }>>,
  ) {
    this.#typeKindEmitOptions.set(typeKind, options);

    return this;
  }

  /**
   * @internal
   */
  [getEmitOptionsForTypeSym](program: Program, type: Type) {
    const direct = this.#typeEmitOptions.get(type);
    if (direct || !$(program).scalar.is(type) || isBuiltIn(program, type)) {
      return direct;
    }

    let currentScalar: Scalar | undefined = type;
    while (currentScalar && !isBuiltIn(program, currentScalar) && !this.#typeEmitOptions.has(currentScalar)) {
      currentScalar = currentScalar?.baseScalar;
    }

    if (!currentScalar) {
      return undefined;
    }

    return this.#typeEmitOptions.get(currentScalar);
  }

  /**
   * @internal
   */
  [getEmitOptionsForTypeKindSym](_program: Program, typeKind: Type["kind"]) {
    return this.#typeKindEmitOptions.get(typeKind);
  }
}

export interface ZodCustomEmitPropsBase<TCustomType extends Type> {
  type: TCustomType;
  default: Children;
  baseSchemaParts: () => Children;
  constraintParts: () => Children;
  descriptionParts: () => Children;
}

export type CustomTypeToProps<TCustomType extends Type> = TCustomType extends ModelProperty
  ? ObjectPropertyProps
  : TCustomType extends EnumMember
    ? Record<string, never>
    : TCustomType extends UnionVariant
      ? Record<string, never>
      : TCustomType extends Model | Scalar | Union | Enum
        ? VarDeclarationProps
        : VarDeclarationProps | ObjectPropertyProps;

export interface ZodCustomEmitReferenceProps<TCustomType extends Type> extends ZodCustomEmitPropsBase<TCustomType> {
  member?: ModelProperty;
  memberParts: () => Children;
}

export interface ZodCustomEmitDeclareProps<TCustomType extends Type> extends ZodCustomEmitPropsBase<TCustomType> {
  Declaration: ComponentDefinition<CustomTypeToProps<TCustomType>>;
  declarationProps: CustomTypeToProps<TCustomType>;
}

export type ZodCustomDeclarationComponent<TCustomType extends Type> = ComponentDefinition<
  ZodCustomEmitDeclareProps<TCustomType>
>;

export type ZodCustomReferenceComponent<TCustomType extends Type> = ComponentDefinition<
  ZodCustomEmitReferenceProps<TCustomType>
>;

export interface ZodCustomEmitOptionsBase<TCustomType extends Type> {
  declare?: ZodCustomDeclarationComponent<TCustomType>;
  reference?: ZodCustomReferenceComponent<TCustomType>;
  noDeclaration?: boolean;
}

export interface ZodOptionsContext {
  customEmit?: ZodCustomEmitOptions;
}

export const ZodOptionsContext: ComponentContext<ZodOptionsContext> = createContext({});

export function useZodOptions(): ZodOptionsContext {
  return useContext(ZodOptionsContext)!;
}

export function getEmitOptionsForType(program: Program, type: Type, options?: ZodCustomEmitOptions) {
  return options?.[getEmitOptionsForTypeSym](program, type);
}

export function getEmitOptionsForTypeKind(program: Program, typeKind: Type["kind"], options?: ZodCustomEmitOptions) {
  return options?.[getEmitOptionsForTypeKindSym](program, typeKind);
}
