import type { AliasDecl } from "../model/index.js";
import { formatType, type FormatCtx } from "./format.js";

export function emitAlias(ctx: FormatCtx, decl: AliasDecl): void {
  const w = ctx.writer;
  w.doc(decl.doc);
  // Use Go's `type X = Y` alias form for true aliases (no method set divergence).
  // Speakeasy doesn't emit aliases — it inlines the target type at each use site.
  // But emitting an alias here is cheaper and preserves intent.
  w.line(`type ${decl.name} = ${formatType(ctx, decl.target)}`);
}
