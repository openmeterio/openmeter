/**
 * Emit a non-discriminated (heuristic) union.
 *
 *   type <Name>Type string
 *   const (<Name>Type<Variant> <Name>Type = "<VariantName>", ...)
 *
 *   type <Name> struct {
 *     <Variant1> *<Variant1Type> `queryParam:"inline" union:"member"`
 *     ...
 *     Type <Name>Type
 *   }
 *
 *   func Create<Name><Variant>(<param> <VariantType>) <Name> { ... }
 *
 *   func (u *<Name>) UnmarshalJSON(data []byte) error {
 *     try each variant with utils.UnmarshalJSON
 *     pick best with utils.PickBestUnionCandidate
 *   }
 *
 *   func (u <Name>) MarshalJSON() ([]byte, error) {
 *     return whichever variant pointer is non-nil
 *   }
 */
import type { HeuristicUnionDecl, GoType } from "../model/index.js";
import { formatType, type FormatCtx } from "./format.js";

export function emitHeuristicUnion(ctx: FormatCtx, decl: HeuristicUnionDecl): void {
  const w = ctx.writer;
  const enumName = `${decl.name}Type`;
  const recv = "u";
  const utilsImport = `${ctx.module}/internal/utils`;

  // Parent enum
  w.line(`type ${enumName} string`);
  w.blankLine();
  w.line("const (");
  w.indent(() => {
    for (const v of decl.variants) {
      w.line(`${enumName}${v.name} ${enumName} = ${JSON.stringify(v.name)}`);
    }
  });
  w.line(")");
  w.blankLine();

  // Struct
  w.doc(decl.doc);
  w.line(`type ${decl.name} struct {`);
  w.indent(() => {
    for (const v of decl.variants) {
      const typeStr = formatType(ctx, v.type);
      w.line(`${v.name} *${typeStr} \`queryParam:"inline" union:"member"\``);
    }
    w.blankLine();
    w.line(`Type ${enumName}`);
  });
  w.line("}");
  w.blankLine();

  // Constructors
  for (const v of decl.variants) {
    const ctorName = `Create${decl.name}${v.name}`;
    const paramName = camel(v.name);
    const variantTypeStr = formatType(ctx, v.type);
    w.line(`func ${ctorName}(${paramName} ${variantTypeStr}) ${decl.name} {`);
    w.indent(() => {
      w.line(`typ := ${enumName}${v.name}`);
      w.blankLine();
      w.line(`return ${decl.name}{`);
      w.indent(() => {
        w.line(`${v.name}: &${paramName},`);
        w.line("Type: typ,");
      });
      w.line("}");
    });
    w.line("}");
    w.blankLine();
  }

  // UnmarshalJSON
  w.import(utilsImport);
  w.import("fmt");
  w.line(`func (${recv} *${decl.name}) UnmarshalJSON(data []byte) error {`);
  w.blankLine();
  w.indent(() => {
    w.line("var candidates []utils.UnionCandidate");
    w.blankLine();
    w.line("// Collect all valid candidates");
    for (const v of decl.variants) {
      const local = safeLocal(camel(v.name));
      const variantTypeStr = formatType(ctx, v.type);
      const zero = candidateZero(ctx, v.type, variantTypeStr);
      w.line(`var ${local} ${variantTypeStr} = ${zero}`);
      w.line(`if err := utils.UnmarshalJSON(data, &${local}, "", true, nil); err == nil {`);
      w.indent(() => {
        w.line("candidates = append(candidates, utils.UnionCandidate{");
        w.indent(() => {
          w.line(`Type:  ${enumName}${v.name},`);
          w.line(`Value: &${local},`);
        });
        w.line("})");
      });
      w.line("}");
      w.blankLine();
    }
    w.line("if len(candidates) == 0 {");
    w.indent(() =>
      w.line(
        `return fmt.Errorf("could not unmarshal \`%s\` into any supported union types for ${decl.name}", string(data))`,
      ),
    );
    w.line("}");
    w.blankLine();
    w.line("best := utils.PickBestUnionCandidate(candidates, data)");
    w.line("if best == nil {");
    w.indent(() =>
      w.line(
        `return fmt.Errorf("could not unmarshal \`%s\` into any supported union types for ${decl.name}", string(data))`,
      ),
    );
    w.line("}");
    w.blankLine();
    w.line(`${recv}.Type = best.Type.(${enumName})`);
    w.line("switch best.Type {");
    for (const v of decl.variants) {
      w.line(`case ${enumName}${v.name}:`);
      w.indent(() => {
        const variantTypeStr = formatType(ctx, v.type);
        w.line(`${recv}.${v.name} = best.Value.(*${variantTypeStr})`);
        w.line("return nil");
      });
    }
    w.line("}");
    w.blankLine();
    w.line(
      `return fmt.Errorf("could not unmarshal \`%s\` into any supported union types for ${decl.name}", string(data))`,
    );
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

function candidateZero(ctx: FormatCtx, t: GoType, formatted: string): string {
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
      return "time.Time{}";
    case "named":
      if (ctx.stringBackedTypes.has(t.name)) return `${formatted}("")`;
      return `${formatted}{}`;
  }
}
