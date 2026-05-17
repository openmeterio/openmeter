/**
 * Emit a Go enum declaration.
 *
 *   type MeterAggregation string
 *
 *   const (
 *       MeterAggregationSum         MeterAggregation = "sum"
 *       ...
 *       MeterAggregationUnknown     MeterAggregation = "UNKNOWN"  // when forward-compatible
 *   )
 *
 *   func (e MeterAggregation) ToPointer() *MeterAggregation { ... }
 *   func (e *MeterAggregation) UnmarshalJSON(data []byte) error { ... }
 *
 * The UnmarshalJSON is permissive (any string is accepted) because Speakeasy
 * generates forward-compatible enums by default.
 */
import type { EnumDecl } from "../model/index.js";
import type { Writer } from "./writer.js";

export function emitEnum(w: Writer, decl: EnumDecl): void {
  w.doc(decl.doc);
  w.line(`type ${decl.name} string`);
  w.blankLine();

  // Const block
  const hasUnknown = decl.members.some((m) => m.name === "Unknown");
  w.line("const (");
  w.indent(() => {
    for (const m of decl.members) {
      // PascalCase enum name + member name → `MeterAggregationSum`.
      const constName = `${decl.name}${m.name}`;
      w.line(`${constName} ${decl.name} = ${JSON.stringify(m.value)}`);
    }
    if (decl.forwardCompatible && !hasUnknown) {
      const unknownName = `${decl.name}Unknown`;
      w.line(`${unknownName} ${decl.name} = "UNKNOWN"`);
    }
  });
  w.line(")");
  w.blankLine();

  // ToPointer convenience
  w.line(`func (e ${decl.name}) ToPointer() *${decl.name} {`);
  w.indent(() => w.line("return &e"));
  w.line("}");
  w.blankLine();

  // UnmarshalJSON — accept any string; matches Speakeasy's forward-compat behavior.
  w.import("encoding/json");
  w.line(`func (e *${decl.name}) UnmarshalJSON(data []byte) error {`);
  w.indent(() => {
    w.line("var v string");
    w.line("if err := json.Unmarshal(data, &v); err != nil {");
    w.indent(() => w.line("return err"));
    w.line("}");
    w.line(`*e = ${decl.name}(v)`);
    w.line("return nil");
  });
  w.line("}");
}
