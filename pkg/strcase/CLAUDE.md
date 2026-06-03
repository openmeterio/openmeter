# strcase

<!-- archie:ai-start -->

> Minimal zero-dependency string-case conversion package providing SnakeToCamel and CamelToSnake for converting JSON field names or database column identifiers to and from Go naming conventions.

## Patterns

**Only underscore is the separator for SnakeToCamel** — SnakeToCamel splits only on '_'; hyphens and slashes pass through unchanged. CamelToSnake splits only on uppercase runes. (`strcase.SnakeToCamel("a_b-c") == "aB-c" // hyphen preserved`)
**First character case preserved** — SnakeToCamel does not uppercase the first character — result starts lowercase (camelCase, not PascalCase). (`strcase.SnakeToCamel("abc_def") == "abcDef" // NOT "AbcDef"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `strcase.go` | Two pure functions: SnakeToCamel and CamelToSnake with round-trip guarantee for well-formed identifiers (letters, digits, underscores, no consecutive underscores). | SnakeToCamel produces camelCase not PascalCase. CamelToSnake does not insert an underscore before a leading uppercase character. |

## Anti-Patterns

- Using this for PascalCase conversion — SnakeToCamel preserves the first character's case.
- Expecting kebab-case or dot-notation handling — only underscore delimiters are understood.

## Decisions

- **Zero external dependencies, hand-rolled implementation.** — Avoids pulling a heavy strcase library for two simple transformations used in generated or utility code.

<!-- archie:ai-end -->
