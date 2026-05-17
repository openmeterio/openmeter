/**
 * Emit a Go struct declaration with:
 *   - tagged fields (json:"name,omitzero" where appropriate)
 *   - MarshalJSON / UnmarshalJSON delegating to vendored utils
 *   - Get<Field>() getter per public field, with nil receiver guard.
 *
 * If `discriminatorVariant` is set on the StructDecl, additionally emits a
 * per-variant singleton enum type (e.g. `TypeFlatFee`) and a `Type` field on
 * the struct.
 */
import type { StructDecl, StructField, GoType } from "../model/index.js";
import { formatType, optionalize, type FormatCtx } from "./format.js";
import { snakeToPascal } from "./naming-helpers.js";
import type { Writer } from "./writer.js";

export function emitStruct(ctx: FormatCtx, decl: StructDecl): void {
  const w = ctx.writer;
  const recv = receiverName(decl.name);
  const utilsImport = `${ctx.module}/internal/utils`;

  // Emit per-variant singleton enum if this is a discriminator variant.
  if (decl.discriminatorVariant) {
    emitVariantSingletonEnum(w, decl.discriminatorVariant.singletonEnumName, decl.discriminatorVariant.value);
    w.blankLine();
  }

  w.doc(decl.doc);
  w.line(`type ${decl.name} struct {`);
  w.indent(() => {
    if (decl.discriminatorVariant) {
      w.line(`Type ${decl.discriminatorVariant.singletonEnumName} \`json:"${getDiscriminatorPropName()}"\``);
    }
    for (const f of decl.fields) {
      // Skip the field if the spec carries it AND we're a variant — the singleton
      // enum is canonical. (For now we don't filter; the spec's `type` field will
      // typically map to a `Type` Go field which collides — emit only one, prefer the variant's.)
      if (decl.discriminatorVariant && f.name === "Type") continue;
      emitField(ctx, f);
    }
  });
  w.line("}");
  w.blankLine();

  // MarshalJSON / UnmarshalJSON
  w.import(utilsImport);
  w.line(`func (${recv} ${decl.name}) MarshalJSON() ([]byte, error) {`);
  w.indent(() => w.line(`return utils.MarshalJSON(${recv}, "", false)`));
  w.line("}");
  w.blankLine();
  w.line(`func (${recv} *${decl.name}) UnmarshalJSON(data []byte) error {`);
  w.indent(() => {
    w.line(`if err := utils.UnmarshalJSON(data, &${recv}, "", false, nil); err != nil {`);
    w.indent(() => w.line("return err"));
    w.line("}");
    w.line("return nil");
  });
  w.line("}");

  // Getters
  if (decl.discriminatorVariant) {
    w.blankLine();
    emitGetter(ctx, recv, decl.name, {
      name: "Type",
      jsonName: "type",
      type: { kind: "named", name: decl.discriminatorVariant.singletonEnumName, pkg: ctx.currentPkg },
      optional: false,
    });
  }
  for (const f of decl.fields) {
    if (decl.discriminatorVariant && f.name === "Type") continue;
    w.blankLine();
    emitGetter(ctx, recv, decl.name, f);
  }
}

function emitField(ctx: FormatCtx, f: StructField): void {
  const w = ctx.writer;
  if (f.doc) {
    for (const ln of f.doc.split("\n")) w.line(`// ${ln}`.trimEnd());
  }
  const fieldType = f.optional ? optionalize(f.type) : f.type;
  const typeStr = formatType(ctx, fieldType);
  const tag = jsonTag(f);
  w.line(`${f.name} ${typeStr} \`${tag}\``);
}

function jsonTag(f: StructField): string {
  const omit = f.optional || isCollectionType(f.type) ? ",omitzero" : "";
  return `json:"${f.jsonName}${omit}"`;
}

function isCollectionType(t: GoType): boolean {
  return t.kind === "slice" || t.kind === "map";
}

function emitGetter(
  ctx: FormatCtx,
  recv: string,
  typeName: string,
  f: StructField,
): void {
  const w = ctx.writer;
  const fieldType = f.optional ? optionalize(f.type) : f.type;
  const typeStr = formatType(ctx, fieldType);
  w.line(`func (${recv} *${typeName}) Get${f.name}() ${typeStr} {`);
  w.indent(() => {
    w.line(`if ${recv} == nil {`);
    w.indent(() => w.line(`return ${zeroValue(ctx, fieldType)}`));
    w.line("}");
    w.line(`return ${recv}.${f.name}`);
  });
  w.line("}");
}

function zeroValue(ctx: FormatCtx, t: GoType): string {
  switch (t.kind) {
    case "scalar":
      if (t.name === "string") return `""`;
      if (t.name === "bool") return "false";
      if (t.name === "[]byte") return "nil";
      return "0";
    case "slice":
    case "map":
    case "pointer":
    case "any":
      return "nil";
    case "time":
      ctx.writer.import("time");
      return "time.Time{}";
    case "named": {
      const formatted = formatType(ctx, t);
      if (ctx.stringBackedTypes.has(t.name)) {
        return `${formatted}("")`;
      }
      return `${formatted}{}`;
    }
  }
}

function emitVariantSingletonEnum(w: Writer, name: string, value: string): void {
  w.line(`type ${name} string`);
  w.blankLine();
  w.line("const (");
  w.indent(() => {
    w.line(`${name}${snakeToPascal(value)} ${name} = ${JSON.stringify(value)}`);
  });
  w.line(")");
}

function receiverName(typeName: string): string {
  // Lowercased first char, matching reference SDK convention.
  return typeName.charAt(0).toLowerCase();
}

function getDiscriminatorPropName(): string {
  // Always "type" for now. If/when the spec uses a different discriminator
  // property name, the variant struct's field-name and tag must be aligned.
  return "type";
}
