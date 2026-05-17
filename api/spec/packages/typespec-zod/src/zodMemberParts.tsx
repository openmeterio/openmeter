import type { ModelProperty } from "@typespec/compiler";
import type { Typekit } from "@typespec/compiler/typekit";
import { useTsp } from "@typespec/emitter-framework";
import { ValueExpression } from "@typespec/emitter-framework/typescript";
import { callPart } from "./utils.jsx";
import { usesBigIntBase } from "./zodBaseSchema.jsx";

export function zodMemberParts(member?: ModelProperty) {
  const { $ } = useTsp();
  return [...optionalParts($, member), ...defaultParts($, member)];
}

function defaultParts($: Typekit, member?: ModelProperty) {
  if (!member?.defaultValue) {
    return [];
  }

  if (
    member.defaultValue.valueKind === "NumericValue" &&
    $.scalar.is(member.type) &&
    usesBigIntBase($, member.type)
  ) {
    const big = member.defaultValue.value.asBigInt();
    if (big !== null) {
      return [callPart("default", `${big}n`)];
    }
  }

  return [callPart("default", [<ValueExpression value={member.defaultValue} />])];
}

function optionalParts(_$: Typekit, member?: ModelProperty) {
  if (!member?.optional) {
    return [];
  }

  return [callPart("optional")];
}
