/**
 * Emit a discriminated union.
 *
 *   type <Name>Type string                            (synthesized only if not external)
 *   const (...)
 *
 *   const <Name>Unknown <EnumName> = "UNKNOWN"         (always, sentinel for Unknown state)
 *
 *   type <Name> struct {
 *     <Variant1> *<Variant1Type> `queryParam:"inline" union:"member"`
 *     ...
 *     UnknownRaw json.RawMessage `json:"-" union:"unknown"`
 *     Type <EnumName>
 *   }
 *
 *   func Create<Name><Variant>(...) <Name> { ... }
 *   func Create<Name>Unknown(raw json.RawMessage) <Name> { ... }
 *
 *   func (u <Name>) GetUnknownRaw() / IsUnknown() / UnmarshalJSON / MarshalJSON
 *
 * When `decl.discriminatorEnumName` is set, the parent enum is assumed to be
 * declared elsewhere; only the union-specific Unknown sentinel is emitted here.
 */
import type { DiscriminatedUnionDecl } from "../model/index.js";
import { formatType, type FormatCtx } from "./format.js";
import { snakeToPascal } from "./naming-helpers.js";

export function emitDiscriminatedUnion(ctx: FormatCtx, decl: DiscriminatedUnionDecl): void {
  const w = ctx.writer;
  const externalEnum = decl.discriminatorEnumName;
  const enumName = externalEnum ?? `${decl.name}Type`;
  const unknownConst = externalEnum
    ? `${decl.name}Unknown`
    : `${enumName}Unknown`;
  const recv = "u";
  const utilsImport = `${ctx.module}/internal/utils`;

  if (!externalEnum) {
    // Synthesized parent enum with const block (the union owns the enum).
    w.line(`type ${enumName} string`);
    w.blankLine();
    w.line("const (");
    w.indent(() => {
      for (const v of decl.variants) {
        const constName = `${enumName}${snakeToPascal(v.value)}`;
        w.line(`${constName} ${enumName} = ${JSON.stringify(v.value)}`);
      }
      w.line(`${unknownConst} ${enumName} = "UNKNOWN"`);
    });
    w.line(")");
    w.blankLine();
  } else {
    // External enum is already declared as its own component. We only need
    // a sentinel for the "unknown variant" state.
    w.line(`const ${unknownConst} ${enumName} = "UNKNOWN"`);
    w.blankLine();
  }

  // Struct
  w.import("encoding/json");
  w.doc(decl.doc);
  w.line(`type ${decl.name} struct {`);
  w.indent(() => {
    for (const v of decl.variants) {
      const typeStr = formatType(ctx, { kind: "named", name: v.typeRef.name, pkg: v.typeRef.pkg });
      w.line(`${v.name} *${typeStr} \`queryParam:"inline" union:"member"\``);
    }
    w.line(`UnknownRaw json.RawMessage \`json:"-" union:"unknown"\``);
    w.blankLine();
    w.line(`Type ${enumName}`);
  });
  w.line("}");
  w.blankLine();

  // Per-variant constructors
  for (const v of decl.variants) {
    const ctorName = `Create${decl.name}${snakeToPascal(v.value)}`;
    const paramName = safeLocal(camel(v.name));
    const variantSingleton = `Type${snakeToPascal(v.value)}`;
    const enumMemberConst = `${enumName}${snakeToPascal(v.value)}`;
    const variantTypeStr = formatType(ctx, { kind: "named", name: v.typeRef.name, pkg: v.typeRef.pkg });
    w.line(`func ${ctorName}(${paramName} ${variantTypeStr}) ${decl.name} {`);
    w.indent(() => {
      w.line(`typ := ${enumMemberConst}`);
      w.blankLine();
      w.line(`typStr := ${variantSingleton}(typ)`);
      w.line(`${paramName}.Type = typStr`);
      w.blankLine();
      w.line(`return ${decl.name}{`);
      w.indent(() => {
        w.line(`${v.name}: &${paramName},`);
        w.line("Type:                 typ,");
      });
      w.line("}");
    });
    w.line("}");
    w.blankLine();
  }

  // Unknown constructor
  w.line(`func Create${decl.name}Unknown(raw json.RawMessage) ${decl.name} {`);
  w.indent(() => {
    w.line(`return ${decl.name}{`);
    w.indent(() => {
      w.line("UnknownRaw: raw,");
      w.line(`Type:       ${unknownConst},`);
    });
    w.line("}");
  });
  w.line("}");
  w.blankLine();

  // GetUnknownRaw / IsUnknown
  w.line(`func (${recv} ${decl.name}) GetUnknownRaw() json.RawMessage {`);
  w.indent(() => w.line(`return ${recv}.UnknownRaw`));
  w.line("}");
  w.blankLine();
  w.line(`func (${recv} ${decl.name}) IsUnknown() bool {`);
  w.indent(() => w.line(`return ${recv}.Type == ${unknownConst}`));
  w.line("}");
  w.blankLine();

  // UnmarshalJSON
  w.import(utilsImport);
  w.import("fmt");
  w.line(`func (${recv} *${decl.name}) UnmarshalJSON(data []byte) error {`);
  w.blankLine();
  w.indent(() => {
    w.line("type discriminator struct {");
    w.indent(() => w.line(`Type string \`json:"${decl.discriminatorProperty}"\``));
    w.line("}");
    w.blankLine();
    w.line("dis := new(discriminator)");
    w.line("if err := json.Unmarshal(data, &dis); err != nil {");
    w.indent(() => {
      w.line(`${recv}.UnknownRaw = json.RawMessage(data)`);
      w.line(`${recv}.Type = ${unknownConst}`);
      w.line("return nil");
    });
    w.line("}");
    w.line("if dis == nil {");
    w.indent(() => {
      w.line(`${recv}.UnknownRaw = json.RawMessage(data)`);
      w.line(`${recv}.Type = ${unknownConst}`);
      w.line("return nil");
    });
    w.line("}");
    w.blankLine();
    w.line("switch dis.Type {");
    for (const v of decl.variants) {
      const enumMemberConst = `${enumName}${snakeToPascal(v.value)}`;
      w.line(`case ${JSON.stringify(v.value)}:`);
      w.indent(() => {
        const local = safeLocal(camel(v.name));
        w.line(`${local} := new(${formatType(ctx, { kind: "named", name: v.typeRef.name, pkg: v.typeRef.pkg })})`);
        w.line(`if err := utils.UnmarshalJSON(data, &${local}, "", true, nil); err != nil {`);
        w.indent(() =>
          w.line(
            `return fmt.Errorf("could not unmarshal \`%s\` into expected (Type == ${v.value}) type ${v.typeRef.name} within ${decl.name}: %w", string(data), err)`,
          ),
        );
        w.line("}");
        w.blankLine();
        w.line(`${recv}.${v.name} = ${local}`);
        w.line(`${recv}.Type = ${enumMemberConst}`);
        w.line("return nil");
      });
    }
    w.line("default:");
    w.indent(() => {
      w.line(`${recv}.UnknownRaw = json.RawMessage(data)`);
      w.line(`${recv}.Type = ${unknownConst}`);
      w.line("return nil");
    });
    w.line("}");
    w.blankLine();
  });
  w.line("}");
  w.blankLine();

  // MarshalJSON
  w.import("errors");
  w.line(`func (${recv} ${decl.name}) MarshalJSON() ([]byte, error) {`);
  w.indent(() => {
    for (const v of decl.variants) {
      w.line(`if ${recv}.${v.name} != nil {`);
      w.indent(() => w.line(`return utils.MarshalJSON(${recv}.${v.name}, "", true)`));
      w.line("}");
      w.blankLine();
    }
    w.line(`if ${recv}.UnknownRaw != nil {`);
    w.indent(() => w.line(`return json.RawMessage(${recv}.UnknownRaw), nil`));
    w.line("}");
    w.line(`return nil, errors.New("could not marshal union type ${decl.name}: all fields are null")`);
  });
  w.line("}");
}

function camel(s: string): string {
  if (!s) return s;
  return s.charAt(0).toLowerCase() + s.slice(1);
}

const GO_KEYWORDS = new Set([
  "break", "case", "chan", "const", "continue", "default", "defer", "else",
  "fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
  "map", "package", "range", "return", "select", "struct", "switch", "type",
  "var",
]);
const GO_PREDECLARED = new Set([
  "any", "bool", "byte", "complex64", "complex128", "error", "float32",
  "float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
  "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
]);

function safeLocal(name: string): string {
  if (GO_KEYWORDS.has(name) || GO_PREDECLARED.has(name)) return `${name}_`;
  return name;
}
