# AIP-3106 — Empty entity fields

Reference: https://kong-aip.netlify.app/aip/3106/

## Always return all fields

Every field defined for an entity must appear in responses, regardless of whether it has a value. This lets clients iterate and access fields without first checking whether they exist.

## Representation of "empty"

| Type   | Empty representation | Notes                                                                                                      |
| ------ | -------------------- | ---------------------------------------------------------------------------------------------------------- |
| Scalar | `null`               | **Not** the language's zero value. `0`, `""`, and `false` are valid non-empty values distinct from `null`. |
| List   | `[]`                 | Lets clients loop without null-check guards.                                                               |
| Object | `{}`                 | Lets clients access attributes without existence guards.                                                   |

## Language gotcha

Some server languages coerce empty / zero values where they shouldn't. Make the absent scalar explicitly `null` on the wire — for example, do not let `0` leak out when the semantic meaning is "unset integer field", since `0` is a distinct valid value.

## Schema variants

When a response payload can differ based on entity configuration (e.g. some subtypes carry extra fields), declare all possible shapes with `oneOf` so the "always return all fields" rule is satisfied per variant rather than across an undifferentiated union.
