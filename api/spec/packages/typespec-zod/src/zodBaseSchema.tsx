import { type Children, For, refkey } from "@alloy-js/core";
import { ArrayExpression, MemberExpression, ObjectExpression, ObjectProperty } from "@alloy-js/typescript";
import type { Enum, LiteralType, Model, Scalar, Tuple, Type, Union } from "@typespec/compiler";
import type { Typekit } from "@typespec/compiler/typekit";
import { useTsp } from "@typespec/emitter-framework";
import { ZodCustomTypeComponent } from "./components/ZodCustomTypeComponent.jsx";
import { ZodSchema } from "./components/ZodSchema.jsx";
import {
  callPart,
  idPart,
  isDeclaration,
  isRecord,
  refkeySym,
  shouldReference,
  subtreeReachesType,
  useDeclaringType,
  zodMemberExpr,
} from "./utils.jsx";

/**
 * Returns the identifier parts for the base Zod schema for a given TypeSpec type.
 */
export function zodBaseSchemaParts(type: Type) {
  const { $ } = useTsp();

  switch (type.kind) {
    case "Intrinsic":
      return intrinsicBaseType(type);
    case "String":
    case "Number":
    case "Boolean":
      return literalBaseType(type);
    case "Scalar":
      return scalarBaseType($, type);
    case "Model":
      return modelBaseType(type);
    case "Union":
      return unionBaseType(type);
    case "Enum":
      return enumBaseType(type);
    case "ModelProperty":
      return zodBaseSchemaParts(type.type);
    case "EnumMember":
      return type.value ? literalBaseType($.literal.create(type.value)) : literalBaseType($.literal.create(type.name));
    case "Tuple":
      return tupleBaseType(type);
    default:
      return zodMemberExpr(callPart("any"));
  }
}

/**
 * Returns true when {@link scalarBaseType} would emit `z.bigint()` for this scalar.
 * Kept colocated with the base-type decision so the constraint emitter renders
 * literal values as bigints (`1n`) rather than numbers (`1`) for matching types.
 */
export function usesBigIntBase($: Typekit, type: Scalar): boolean {
  if (!$.scalar.extendsInteger(type)) {
    return false;
  }
  return !(
    $.scalar.extendsInt32(type) ||
    $.scalar.extendsUint32(type) ||
    $.scalar.extendsSafeint(type)
  );
}

function literalBaseType(type: LiteralType) {
  switch (type.kind) {
    case "String":
      return zodMemberExpr(callPart("literal", `"${type.value}"`));
    case "Number":
    case "Boolean":
      return zodMemberExpr(callPart("literal", `${type.value}`));
  }
}

function scalarBaseType($: Typekit, type: Scalar) {
  if (type.baseScalar && shouldReference($.program, type.baseScalar)) {
    return <MemberExpression.Part refkey={refkey(type.baseScalar, refkeySym)} />;
  }

  if ($.scalar.extendsBoolean(type)) {
    return zodMemberExpr(callPart("boolean"));
  }

  if ($.scalar.extendsNumeric(type)) {
    if ($.scalar.extendsInteger(type)) {
      if (usesBigIntBase($, type)) {
        return zodMemberExpr(callPart("bigint"));
      }
      return zodMemberExpr(callPart("number"), callPart("int"));
    }
    // floats and such; lacking a decimal type this is the best we can do.
    return zodMemberExpr(callPart("number"));
  }

  if ($.scalar.extendsString(type)) {
    if ($.scalar.extendsUrl(type)) {
      return zodMemberExpr(callPart("string"), callPart("url"));
    }
    return zodMemberExpr(callPart("string"));
  }

  if ($.scalar.extendsBytes(type)) {
    return zodMemberExpr(callPart("any"));
  }

  if ($.scalar.extendsPlainDate(type)) {
    return zodMemberExpr(idPart("coerce"), callPart("date"));
  }

  if ($.scalar.extendsPlainTime(type)) {
    return zodMemberExpr(callPart("string"), callPart("time"));
  }

  if ($.scalar.extendsUtcDateTime(type)) {
    const encoding = $.scalar.getEncoding(type);
    if (encoding === undefined) {
      return zodMemberExpr(idPart("coerce"), callPart("date"));
    }
    if (encoding.encoding === "unixTimestamp") {
      return scalarBaseType($, encoding.type);
    }
    if (encoding.encoding === "rfc3339") {
      return zodMemberExpr(callPart("string"), callPart("datetime"));
    }
    return scalarBaseType($, encoding.type);
  }

  if ($.scalar.extendsOffsetDateTime(type)) {
    const encoding = $.scalar.getEncoding(type);
    if (encoding === undefined) {
      return zodMemberExpr(idPart("coerce"), callPart("date"));
    }
    if (encoding.encoding === "rfc3339") {
      return zodMemberExpr(callPart("string"), callPart("datetime"));
    }
    return scalarBaseType($, encoding.type);
  }

  if ($.scalar.extendsDuration(type)) {
    const encoding = $.scalar.getEncoding(type);
    if (encoding === undefined || encoding.encoding === "ISO8601") {
      return zodMemberExpr(callPart("string"), callPart("duration"));
    }
    return scalarBaseType($, encoding.type);
  }

  return zodMemberExpr(callPart("any"));
}

function enumBaseType(type: Enum) {
  return zodMemberExpr(
    callPart(
      "enum",
      <ArrayExpression>
        <For each={type.members.values()} comma line>
          {(member) => (
            <ZodCustomTypeComponent
              type={member}
              Declaration={(props: { children?: Children }) => props.children}
              declarationProps={{}}
              declare
            >
              {JSON.stringify(member.value ?? member.name)}
            </ZodCustomTypeComponent>
          )}
        </For>
      </ArrayExpression>,
    ),
  );
}

function tupleBaseType(type: Tuple) {
  return zodMemberExpr(
    callPart(
      "tuple",
      <ArrayExpression>
        <For each={type.values} comma line>
          {(item) => <ZodSchema type={item} nested />}
        </For>
      </ArrayExpression>,
    ),
  );
}

function modelBaseType(type: Model) {
  const { $ } = useTsp();

  if ($.array.is(type)) {
    return zodMemberExpr(callPart("array", <ZodSchema type={type.indexer?.value} nested />));
  }

  let recordPart: Children;
  if (
    isRecord($.program, type) ||
    (!!type.baseModel && isRecord($.program, type.baseModel) && !isDeclaration($.program, type.baseModel))
  ) {
    const indexer = (type.indexer ?? type.baseModel?.indexer)!;
    recordPart = zodMemberExpr(
      callPart("record", <ZodSchema type={indexer.key} nested />, <ZodSchema type={indexer.value} nested />),
    );
  }

  const declaringType = useDeclaringType();
  let memberPart: Children;
  if (type.properties.size > 0) {
    const members = (
      <ObjectExpression>
        <For each={type.properties.values()} comma hardline enderPunctuation>
          {(prop) => {
            const isCyclic =
              declaringType !== undefined && subtreeReachesType($.program, prop.type, declaringType);
            const propertyContent = isCyclic ? (
              <>
                get {prop.name}() {"{ return "}
                <ZodSchema type={prop} nested />
                {"; }"}
              </>
            ) : (
              <ObjectProperty name={prop.name}>
                <ZodSchema type={prop} nested />
              </ObjectProperty>
            );
            return (
              <ZodCustomTypeComponent
                type={prop}
                declare
                Declaration={ObjectProperty}
                declarationProps={{ name: prop.name }}
              >
                {propertyContent}
              </ZodCustomTypeComponent>
            );
          }}
        </For>
      </ObjectExpression>
    );
    memberPart = zodMemberExpr(callPart("object", members));
  }

  let parts: Children;

  if (!memberPart && !recordPart) {
    parts = zodMemberExpr(callPart("object", <ObjectExpression />));
  } else if (memberPart && recordPart) {
    parts = zodMemberExpr(callPart("intersection", memberPart, recordPart));
  } else {
    parts = memberPart ?? recordPart;
  }

  if (type.baseModel && shouldReference($.program, type.baseModel)) {
    return (
      <MemberExpression>
        <MemberExpression.Part refkey={refkey(type.baseModel, refkeySym)} />
        <MemberExpression.Part id="merge" />
        <MemberExpression.Part args={[parts]} />
      </MemberExpression>
    );
  }

  return parts;
}

function unionBaseType(type: Union) {
  const { $ } = useTsp();

  const discriminated = $.union.getDiscriminatedUnion(type);

  if ($.union.isExpression(type) || !discriminated) {
    return zodMemberExpr(
      callPart(
        "union",
        <ArrayExpression>
          <For each={type.variants} comma line>
            {(_name, variant) => <ZodSchema type={variant.type} nested />}
          </For>
        </ArrayExpression>,
      ),
    );
  }

  const propKey = discriminated.options.discriminatorPropertyName;
  const envKey = discriminated.options.envelopePropertyName;
  const unionArgs = [
    `"${propKey}"`,
    <ArrayExpression>
      <For each={Array.from(type.variants.values())} comma line>
        {(variant) => {
          if (discriminated.options.envelope === "object") {
            const envelope = $.model.create({
              properties: {
                [propKey]: $.modelProperty.create({
                  name: propKey,
                  type: $.literal.create(variant.name as string),
                }),
                [envKey]: $.modelProperty.create({
                  name: envKey,
                  type: variant.type,
                }),
              },
            });
            return <ZodSchema type={envelope} nested />;
          }
          return <ZodSchema type={variant.type} nested />;
        }}
      </For>
    </ArrayExpression>,
  ];

  return zodMemberExpr(callPart("discriminatedUnion", ...unionArgs));
}

function intrinsicBaseType(type: Type) {
  if (type.kind === "Intrinsic") {
    switch (type.name) {
      case "null":
        return zodMemberExpr(callPart("null"));
      case "never":
        return zodMemberExpr(callPart("never"));
      case "unknown":
        return zodMemberExpr(callPart("unknown"));
      case "void":
        return zodMemberExpr(callPart("void"));
      default:
        return zodMemberExpr(callPart("any"));
    }
  }
  return zodMemberExpr(callPart("any"));
}
