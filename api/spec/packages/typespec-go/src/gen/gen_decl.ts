/**
 * Emit a single Decl as its own .go file under models/components/.
 *
 * Returns a complete file body string (ready to write to disk).
 */
import type { Decl } from "../model/index.js";
import { emitAlias } from "./gen_alias.js";
import { emitDiscriminatedUnion } from "./gen_disc_union.js";
import { emitEnum } from "./gen_enum.js";
import { emitHeuristicUnion } from "./gen_heur_union.js";
import { emitStruct } from "./gen_struct.js";
import type { FormatCtx } from "./format.js";
import { GENERATED_BANNER, Writer } from "./writer.js";

export interface DeclFile {
  /** Path relative to the emitter output dir (e.g. "models/components/meter.go"). */
  readonly path: string;
  readonly content: string;
}

const COMPONENTS_PKG = "models/components";

export function emitDeclFile(
  module: string,
  decl: Decl,
  stringBackedTypes: ReadonlySet<string>,
): DeclFile {
  const w = new Writer();
  // Generation banner sits above `package`; `go vet` / IDEs match it there.
  w.preamble(GENERATED_BANNER);
  w.packageName("components");
  const ctx: FormatCtx = { writer: w, module, currentPkg: COMPONENTS_PKG, stringBackedTypes };

  switch (decl.kind) {
    case "struct":
      emitStruct(ctx, decl);
      break;
    case "enum":
      emitEnum(w, decl);
      break;
    case "discriminated-union":
      emitDiscriminatedUnion(ctx, decl);
      break;
    case "heuristic-union":
      emitHeuristicUnion(ctx, decl);
      break;
    case "alias":
      emitAlias(ctx, decl);
      break;
  }

  // File name: lowercase concatenation of the decl name + .go
  const fileName = `${decl.name.toLowerCase()}.go`;
  return {
    path: `${COMPONENTS_PKG}/${fileName}`,
    content: w.finish(),
  };
}
