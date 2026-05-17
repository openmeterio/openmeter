/**
 * Go naming conventions.
 *
 * The reference SDK follows these rules:
 *   - Exported identifiers are PascalCase.
 *   - JSON field names are snake_case (preserved from the spec).
 *   - File names are lowercased concatenation (e.g. `BillingFlatFeeCharge` -> `billingflatfeecharge.go`).
 *   - Reserved words get an underscore suffix.
 */

const GO_RESERVED_WORDS = new Set([
  "break", "case", "chan", "const", "continue", "default", "defer", "else",
  "fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
  "map", "package", "range", "return", "select", "struct", "switch", "type", "var",
]);

const GO_BUILTIN_TYPES = new Set([
  "any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64",
  "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8",
  "uint16", "uint32", "uint64", "uintptr",
]);

/** Convert any identifier to PascalCase. Splits on `_`, `-`, and case boundaries. */
export function pascal(input: string): string {
  return splitWords(input)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join("");
}

/** Convert any identifier to camelCase. */
export function camel(input: string): string {
  const p = pascal(input);
  return p.charAt(0).toLowerCase() + p.slice(1);
}

/** Convert to the all-lowercase file-base form used in the reference SDK (e.g. `BillingFlatFeeCharge` -> `billingflatfeecharge`). */
export function fileBase(input: string): string {
  return splitWords(input).join("").toLowerCase();
}

/** Escape a Go reserved word by appending an underscore. */
export function escapeReserved(name: string): string {
  if (GO_RESERVED_WORDS.has(name) || GO_BUILTIN_TYPES.has(name)) {
    return name + "_";
  }
  return name;
}

function splitWords(input: string): string[] {
  if (!input) return [];
  // Split on non-alphanumeric runs, then on lower→upper case boundaries.
  return input
    .replace(/[^a-zA-Z0-9]+/g, " ")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .trim()
    .split(/\s+/)
    .filter((w) => w.length > 0);
}
