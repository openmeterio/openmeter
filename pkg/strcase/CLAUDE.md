# strcase

<!-- archie:ai-start -->

> Minimal string-case conversion utilities (snake_case ↔ camelCase) with no external dependencies, used wherever JSON field names or database column identifiers need to be converted to/from Go naming conventions.

## Patterns

**Only underscore is treated as word separator** — SnakeToCamel only splits on '_'; hyphens and slashes pass through unchanged. CamelToSnake only splits on uppercase runes. (`strcase.SnakeToCamel("a_b-c") == "aB-c"`)
**Round-trip guarantee for well-formed identifiers** — CamelToSnake(SnakeToCamel(x)) == x and SnakeToCamel(CamelToSnake(x)) == x for identifiers that contain only letters, digits, underscores, and no consecutive underscores. (`strcase.SnakeToCamel(strcase.CamelToSnake("abcDef")) == "abcDef"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `strcase.go` | Two pure functions: SnakeToCamel and CamelToSnake | First character is not uppercased by SnakeToCamel — result starts lowercase; CamelToSnake does not insert underscore before the first character even if it is uppercase |

## Anti-Patterns

- Using this package for PascalCase conversion — SnakeToCamel preserves the first character's case
- Expecting kebab-case or dot-notation to be handled — only underscore delimiters are understood

## Decisions

- **Zero external dependencies, hand-rolled implementation** — Avoids pulling in a heavy strcase library for two simple transformations used in generated or utility code

<!-- archie:ai-end -->
