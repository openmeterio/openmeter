import type { Children } from "@alloy-js/core";
import { getFormat, getPattern, type ModelProperty, type Scalar, type Type } from "@typespec/compiler";
import type { Typekit } from "@typespec/compiler/typekit";
import { useTsp } from "@typespec/emitter-framework";
import { callPart, shouldReference } from "./utils.jsx";
import { usesBigIntBase } from "./zodBaseSchema.jsx";

export function zodConstraintsParts(type: Type, member?: ModelProperty) {
  const { $ } = useTsp();

  if ($.scalar.extendsNumeric(type)) {
    return numericConstraintsParts($, type, member);
  }

  if ($.scalar.extendsString(type)) {
    return stringConstraints($, type, member);
  }

  if ($.scalar.extendsUtcDateTime(type) || $.scalar.extendsOffsetDateTime(type) || $.scalar.extendsDuration(type)) {
    const encoding = $.scalar.getEncoding(type);
    if (encoding === undefined) {
      return [];
    }
    return numericConstraintsToParts(
      intrinsicNumericConstraints($, encoding.type),
      $.scalar.is(encoding.type) && usesBigIntBase($, encoding.type),
    );
  }

  if ($.array.is(type)) {
    return arrayConstraints($, type, member);
  }

  return [];
}

interface StringConstraints {
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  format?: string;
}

function stringConstraints($: Typekit, type: Scalar, member?: ModelProperty) {
  const sources = getDecoratorSources($, type, member);
  const constraints: StringConstraints = {};
  for (const source of sources.reverse()) {
    assignStringConstraints(constraints, {
      minLength: $.type.minLength(source),
      maxLength: $.type.maxLength(source),
      pattern: getPattern($.program, source),
      format: getFormat($.program, source),
    });
  }

  const parts: Children[] = [];

  for (const [name, value] of Object.entries(constraints)) {
    if (value === undefined) {
      continue;
    }
    if (name === "minLength" && value !== 0) {
      parts.push(callPart("min", value));
    } else if (name === "maxLength" && Number.isFinite(value)) {
      parts.push(callPart("max", value));
    } else if (name === "pattern") {
      parts.push(callPart("regex", `/${value}/`));
    } else if (name === "format") {
      const method = zodStringFormatMethod(value as string);
      if (method) {
        parts.push(callPart(method));
      }
    }
  }

  return parts;
}

function assignStringConstraints(target: StringConstraints, source: StringConstraints) {
  target.minLength = maxNumeric(target.minLength, source.minLength);
  target.maxLength = minNumeric(target.maxLength, source.maxLength);
  target.pattern = target.pattern ?? source.pattern;
  target.format = target.format ?? source.format;
}

interface NumericConstraints {
  min?: number | bigint;
  max?: number | bigint;
  minExclusive?: number | bigint;
  maxExclusive?: number | bigint;
  safe?: boolean;
}

function maxNumeric<T extends number | bigint>(...values: (T | undefined)[]): T | undefined {
  const definedValues = values.filter((v): v is T => v !== undefined);

  if (definedValues.length === 0) {
    return undefined;
  }

  return definedValues.reduce((max, current) => (current > (max ?? -Infinity) ? current : max), definedValues[0]);
}

function minNumeric<T extends number | bigint>(...values: (T | undefined)[]): T | undefined {
  const definedValues = values.filter((v): v is T => v !== undefined);

  if (definedValues.length === 0) {
    return undefined;
  }

  return definedValues.reduce((min, current) => (current < (min ?? Infinity) ? current : min), definedValues[0]);
}

/**
 * Return sources from most specific to least specific.
 */
function getDecoratorSources<T extends Type>($: Typekit, type: T, member?: ModelProperty): (T | ModelProperty)[] {
  if (!$.scalar.is(type)) {
    return [...(member ? [member] : []), type];
  }

  const sources: (Scalar | ModelProperty)[] = [...(member ? [member] : []), type];

  let currentType: Scalar | undefined = type.baseScalar;
  while (currentType && !shouldReference($.program, currentType)) {
    sources.push(currentType);
    currentType = currentType.baseScalar;
  }
  return sources as (T | ModelProperty)[];
}

function numericConstraintsParts($: Typekit, type: Scalar, member?: ModelProperty) {
  const sources = getDecoratorSources($, type, member);
  const intrinsicConstraints = intrinsicNumericConstraints($, type);
  const decoratorConstraints = decoratorNumericConstraints($, sources);

  // Decorator-internal: if both inclusive and exclusive bounds are set on the
  // same side, keep whichever is tighter and drop the looser one.
  if (decoratorConstraints.min !== undefined && decoratorConstraints.minExclusive !== undefined) {
    if (decoratorConstraints.minExclusive > decoratorConstraints.min) {
      decoratorConstraints.min = undefined;
    } else {
      decoratorConstraints.minExclusive = undefined;
    }
  }

  if (decoratorConstraints.max !== undefined && decoratorConstraints.maxExclusive !== undefined) {
    if (decoratorConstraints.maxExclusive < decoratorConstraints.max) {
      decoratorConstraints.max = undefined;
    } else {
      decoratorConstraints.maxExclusive = undefined;
    }
  }

  // Intrinsic vs decorator: prefer the tighter bound, drop the looser one.
  if (intrinsicConstraints.min !== undefined) {
    if (decoratorConstraints.min !== undefined) {
      if (intrinsicConstraints.min > decoratorConstraints.min) {
        decoratorConstraints.min = undefined;
      } else {
        intrinsicConstraints.min = undefined;
      }
    } else if (decoratorConstraints.minExclusive !== undefined) {
      if (intrinsicConstraints.min > decoratorConstraints.minExclusive) {
        decoratorConstraints.minExclusive = undefined;
      } else {
        intrinsicConstraints.min = undefined;
      }
    }
  }

  if (intrinsicConstraints.max !== undefined) {
    if (decoratorConstraints.max !== undefined) {
      if (intrinsicConstraints.max < decoratorConstraints.max) {
        decoratorConstraints.max = undefined;
      } else {
        intrinsicConstraints.max = undefined;
      }
    } else if (decoratorConstraints.maxExclusive !== undefined) {
      if (intrinsicConstraints.max < decoratorConstraints.maxExclusive) {
        decoratorConstraints.maxExclusive = undefined;
      } else {
        intrinsicConstraints.max = undefined;
      }
    }
  }

  const finalConstraints: NumericConstraints = {};
  assignNumericConstraints(finalConstraints, intrinsicConstraints);
  assignNumericConstraints(finalConstraints, decoratorConstraints);

  return numericConstraintsToParts(finalConstraints, usesBigIntBase($, type));
}

function numericConstraintsToParts(constraints: NumericConstraints, useBigInt: boolean): Children[] {
  const parts: Children[] = [];

  if (constraints.safe) {
    parts.push(callPart("safe"));
  }

  for (const [name, value] of Object.entries(constraints)) {
    if (value === undefined || (typeof value !== "bigint" && !Number.isFinite(value))) {
      continue;
    }
    if (name === "safe") {
      continue;
    }

    if (name === "min" && (value === 0 || value === 0n)) {
      parts.push(callPart("nonnegative"));
      continue;
    }
    const literal = useBigInt || typeof value === "bigint" ? `${BigInt(value as number | bigint)}n` : `${value}`;
    parts.push(callPart(zodNumericConstraintName(name), literal));
  }

  return parts;
}

/**
 * Map a JSON Schema `format` keyword to the corresponding ZodString method.
 * Returns undefined for unknown formats so the emitter skips them.
 */
function zodStringFormatMethod(format: string): string | undefined {
  switch (format) {
    case "uri":
    case "url":
      return "url";
    case "email":
    case "uuid":
    case "ulid":
    case "cuid":
    case "cuid2":
    case "nanoid":
    case "ipv4":
    case "ipv6":
    case "base64":
    case "emoji":
    case "jwt":
      return format;
    case "date-time":
      return "datetime";
    case "date":
      return "date";
    case "time":
      return "time";
    case "duration":
      return "duration";
    case "ip":
      return "ip";
    default:
      return undefined;
  }
}

function zodNumericConstraintName(name: string) {
  if (name === "min") {
    return "gte";
  }
  if (name === "max") {
    return "lte";
  }
  if (name === "minExclusive") {
    return "gt";
  }
  if (name === "maxExclusive") {
    return "lt";
  }
  throw new Error(`Unknown constraint name: ${name}`);
}

function intrinsicNumericConstraints($: Typekit, type: Scalar): NumericConstraints {
  const knownType = $.scalar.getStdBase(type);
  if (!knownType) {
    return {};
  }
  if (!$.scalar.extendsNumeric(knownType)) {
    return {};
  }
  if ($.scalar.extendsSafeint(knownType)) {
    return { safe: true };
  }
  if ($.scalar.extendsInt8(knownType)) {
    return { min: -(1 << 7), max: (1 << 7) - 1 };
  }
  if ($.scalar.extendsInt16(knownType)) {
    return { min: -(1 << 15), max: (1 << 15) - 1 };
  }
  if ($.scalar.extendsInt32(knownType)) {
    return { min: Number(-(1n << 31n)), max: Number((1n << 31n) - 1n) };
  }
  if ($.scalar.extendsInt64(knownType)) {
    return { min: -(1n << 63n), max: (1n << 63n) - 1n };
  }
  if ($.scalar.extendsUint8(knownType)) {
    return { min: 0, max: (1 << 8) - 1 };
  }
  if ($.scalar.extendsUint16(knownType)) {
    return { min: 0, max: (1 << 16) - 1 };
  }
  if ($.scalar.extendsUint32(knownType)) {
    return { min: 0, max: Number((1n << 32n) - 1n) };
  }
  if ($.scalar.extendsUint64(knownType)) {
    return { min: 0n, max: (1n << 64n) - 1n };
  }
  if ($.scalar.extendsFloat32(knownType)) {
    return { min: -3.4028235e38, max: 3.4028235e38 };
  }

  return {};
}

function decoratorNumericConstraints($: Typekit, sources: Type[]) {
  const finalConstraints: NumericConstraints = {};
  for (const source of sources) {
    assignNumericConstraints(finalConstraints, {
      max: $.type.maxValue(source),
      maxExclusive: $.type.maxValueExclusive(source),
      min: $.type.minValue(source),
      minExclusive: $.type.minValueExclusive(source),
    });
  }

  return finalConstraints;
}

function assignNumericConstraints(target: NumericConstraints, source: NumericConstraints) {
  target.min = maxNumeric(target.min, source.min);
  target.max = minNumeric(target.max, source.max);
  target.minExclusive = maxNumeric(source.minExclusive, target.minExclusive);
  target.maxExclusive = minNumeric(source.maxExclusive, target.maxExclusive);
  target.safe = target.safe ?? source.safe;
}

interface ArrayConstraints {
  minItems?: number;
  maxItems?: number;
}

function arrayConstraints($: Typekit, type: Type, member?: ModelProperty) {
  const constraints: ArrayConstraints = {
    minItems: $.type.minItems(type),
    maxItems: $.type.maxItems(type),
  };
  const memberConstraints: ArrayConstraints = {
    minItems: member && $.type.minItems(member),
    maxItems: member && $.type.maxItems(member),
  };

  assignArrayConstraints(constraints, memberConstraints);

  const parts: Children[] = [];

  if (constraints.minItems && constraints.minItems > 0) {
    parts.push(callPart("min", constraints.minItems));
  }

  if (constraints.maxItems && constraints.maxItems > 0) {
    parts.push(callPart("max", constraints.maxItems));
  }

  return parts;
}

function assignArrayConstraints(target: ArrayConstraints, source: ArrayConstraints) {
  target.minItems = maxNumeric(target.minItems, source.minItems);
  target.maxItems = minNumeric(target.maxItems, source.maxItems);
}
