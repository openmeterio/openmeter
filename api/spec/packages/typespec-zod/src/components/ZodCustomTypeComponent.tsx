import type { Children, ComponentDefinition } from "@alloy-js/core";
import type { ModelProperty, Type } from "@typespec/compiler";
import { useTsp } from "@typespec/emitter-framework";
import { getEmitOptionsForType, getEmitOptionsForTypeKind, useZodOptions } from "../context/zod-options.js";
import { zodBaseSchemaParts } from "../zodBaseSchema.jsx";
import { zodConstraintsParts } from "../zodConstraintsParts.jsx";
import { zodDescriptionParts } from "../zodDescriptionParts.jsx";
import { zodMemberParts } from "../zodMemberParts.jsx";

export interface ZodCustomTypeComponentCommonProps<T extends Type> {
  type: T;
  children: Children;
}

export interface ZodCustomTypeComponentDeclarationProps<
  T extends Type,
  // biome-ignore lint/suspicious/noExplicitAny: matches @alloy-js/core ComponentDefinition surface
  U extends ComponentDefinition<any>,
> extends ZodCustomTypeComponentCommonProps<T> {
  declare: true;
  declarationProps: U extends ComponentDefinition<infer P> ? P : never;
  Declaration: U;
}

export interface ZodCustomTypeComponentReferenceProps<T extends Type> extends ZodCustomTypeComponentCommonProps<T> {
  reference: true;
  member?: ModelProperty;
}

export type ZodCustomTypeComponentProps<
  T extends Type,
  // biome-ignore lint/suspicious/noExplicitAny: matches @alloy-js/core ComponentDefinition surface
  U extends ComponentDefinition<any>,
> = ZodCustomTypeComponentDeclarationProps<T, U> | ZodCustomTypeComponentReferenceProps<T>;

export function ZodCustomTypeComponent<
  T extends Type,
  // biome-ignore lint/suspicious/noExplicitAny: matches @alloy-js/core ComponentDefinition surface
  U extends ComponentDefinition<any>,
>(props: ZodCustomTypeComponentProps<T, U>) {
  const options = useZodOptions();
  const { $ } = useTsp();
  const descriptor =
    getEmitOptionsForType($.program, props.type, options.customEmit) ??
    getEmitOptionsForTypeKind($.program, props.type.kind, options.customEmit);

  if (!descriptor) {
    return <>{props.children}</>;
  }

  if ("declare" in props && props.declare && descriptor.declare) {
    const CustomComponent = descriptor.declare;
    return (
      <CustomComponent
        type={props.type}
        default={props.children}
        baseSchemaParts={() => zodBaseSchemaParts(props.type)}
        constraintParts={() => zodConstraintsParts(props.type)}
        descriptionParts={() => zodDescriptionParts(props.type)}
        declarationProps={props.declarationProps}
        Declaration={props.Declaration}
      />
    );
  }

  if ("reference" in props && props.reference && descriptor.reference) {
    const CustomComponent = descriptor.reference;
    return (
      <CustomComponent
        type={props.type}
        member={props.member}
        default={props.children}
        baseSchemaParts={() => zodBaseSchemaParts(props.member ?? props.type)}
        constraintParts={() => zodConstraintsParts(props.type, props.member)}
        descriptionParts={() => zodDescriptionParts(props.type, props.member)}
        memberParts={() => zodMemberParts(props.member)}
      />
    );
  }

  return <>{props.children}</>;
}
