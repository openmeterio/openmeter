# strcase

<!-- archie:ai-start -->

> Minimal snake_case<->camelCase string conversion, used by entitlement/credit/productcatalog HTTP drivers and tools/migrate/viewgen for identifier translation.

## Patterns

**Pure string conversion helpers** — SnakeToCamel uppercases the char after each '_'; CamelToSnake inserts '_' before each uppercase rune and lowercases it. No allocation strategy beyond strings.Builder in CamelToSnake. (`strcase.SnakeToCamel("a_b_c") // "aBC"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `strcase.go` | SnakeToCamel and CamelToSnake functions. | Round-trips only for lowercase-snake input. Non-alphanumerics ('-','/') pass through. Existing uppercase in snake input is preserved, so CamelToSnake(SnakeToCamel(x)) is not guaranteed identity for mixed-case input — see TestCamelToSnakeToCamel covering only well-formed cases. |
| `strcase_test.go` | Table tests for both directions plus the round-trip test. | Test cases include special chars; treat them as the spec for edge behavior. |

## Anti-Patterns

- Assuming a perfect round trip for arbitrary mixed-case input.
- Reaching for a heavier external strcase library when these two functions already cover the API/identifier needs.

## Decisions

- **Hand-rolled instead of a dependency.** — Only two simple conversions are needed (API field <-> identifier casing), keeping the dependency surface minimal.

## Example: Convert an API field name to a DB-style column

```
import "github.com/openmeterio/openmeter/pkg/strcase"

col := strcase.CamelToSnake("createdAt") // "created_at"
```

<!-- archie:ai-end -->
