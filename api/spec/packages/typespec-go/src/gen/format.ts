/**
 * Format a GoType as its on-the-page Go syntax, registering any necessary
 * imports on the supplied Writer.
 */
import type { GoType } from "../model/index.js";
import type { Writer } from "./writer.js";

export interface FormatCtx {
  /** Current Go package the type is being emitted into. */
  readonly currentPkg: string;
  /** Module path (so we can build absolute import paths for cross-package types). */
  readonly module: string;
  readonly writer: Writer;
  /**
   * Names of declared types that are string-backed enums (or aliases of string).
   * Zero-value emission uses `Name("")` for these and `Name{}` otherwise.
   */
  readonly stringBackedTypes: ReadonlySet<string>;
}

export function formatType(ctx: FormatCtx, t: GoType): string {
  switch (t.kind) {
    case "scalar":
      return t.name;
    case "slice":
      return `[]${formatType(ctx, t.element)}`;
    case "map":
      return `map[${formatType(ctx, t.key)}]${formatType(ctx, t.value)}`;
    case "pointer":
      return `*${formatType(ctx, t.element)}`;
    case "time":
      ctx.writer.import("time");
      return "time.Time";
    case "any":
      return "any";
    case "named": {
      const localName = t.name;
      if (t.pkg === ctx.currentPkg) {
        return localName;
      }
      const importPath = `${ctx.module}/${t.pkg}`;
      ctx.writer.import(importPath);
      const pkgAlias = pkgAliasFor(t.pkg);
      return `${pkgAlias}.${localName}`;
    }
  }
}

/**
 * Compute the Go package identifier used in qualified references.
 *
 * Convention from the reference SDK: last path segment of the import path,
 * lowercased, hyphens stripped. e.g. "models/components" -> "components".
 */
export function pkgAliasFor(pkg: string): string {
  const segments = pkg.split("/").filter(Boolean);
  const last = segments[segments.length - 1] ?? "";
  return last.replace(/[^a-zA-Z0-9_]/g, "").toLowerCase();
}

/**
 * Wrap an optional type in `*`. Pointers, slices, and maps are not re-wrapped
 * (they already represent nil-able values in Go).
 */
export function optionalize(t: GoType): GoType {
  if (t.kind === "pointer" || t.kind === "slice" || t.kind === "map") return t;
  return { kind: "pointer", element: t };
}
