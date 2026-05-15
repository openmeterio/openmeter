# strcase

<!-- archie:ai-start -->

> Minimal zero-dependency string-case conversion package providing SnakeToCamel and CamelToSnake for converting JSON field names or database column identifiers to and from Go naming conventions.

## Patterns

**Only underscore is treated as word separator for SnakeToCamel** — SnakeToCamel only splits on '_'; hyphens and slashes pass through unchanged. CamelToSnake only splits on uppercase runes. (`strcase.SnakeToCamel("a_b-c") == "aB-c"  // hyphen preserved`)
**First character case is preserved, not forced to upper** — SnakeToCamel does not uppercase the first character — result starts lowercase. This is camelCase, not PascalCase. (`strcase.SnakeToCamel("abc_def") == "abcDef"  // NOT "AbcDef"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `strcase.go` | Two pure functions: SnakeToCamel and CamelToSnake with round-trip guarantee for well-formed identifiers (letters, digits, underscores, no consecutive underscores). | SnakeToCamel preserves the first character's case — it produces camelCase, not PascalCase. CamelToSnake does not insert underscore before the first character even if it is uppercase. |

## Anti-Patterns

- Using this package for PascalCase conversion — SnakeToCamel preserves the first character's case
- Expecting kebab-case or dot-notation to be handled — only underscore delimiters are understood by SnakeToCamel

## Decisions

- **Zero external dependencies, hand-rolled implementation** — Avoids pulling in a heavy strcase library for two simple transformations used in generated or utility code.

<!-- archie:ai-end -->
