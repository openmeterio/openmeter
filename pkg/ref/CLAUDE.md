# ref

<!-- archie:ai-start -->

> Tiny value-type package providing IDOrKey — a reference that is either a ULID id or a string key — used widely across billing, ledger, and productcatalog to resolve resources by id-or-key.

## Patterns

**ULID-detection parsing** — ParseIDOrKey(s) tries ulid.Parse; success means it's an ID, failure means it's a Key. (`if _, err := ulid.Parse(s); err != nil { n.Key = s } else { n.ID = s }`)
**Validate requires one of id/key** — IDOrKey.Validate() errors when both ID and Key are empty. (`if i.ID == "" && i.Key == "" { return fmt.Errorf("either id or key is required") }`)
**GetIDs/GetKeys nil-safe accessors** — GetIDs/GetKeys return nil (not empty slice) when the respective field is empty, for easy spread into filter args. (`func (i IDOrKey) GetIDs() []string { if i.ID == "" { return nil }; return []string{i.ID} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ref.go` | Defines IDOrKey{ID,Key} with json tags, Validate, GetIDs, GetKeys, and ParseIDOrKey. | Classification is purely ULID-shape based — a key that happens to be a valid ULID will be treated as an ID. Both fields can be set when constructed directly (e.g. from JSON), but ParseIDOrKey sets exactly one. |

## Anti-Patterns

- Assuming a non-empty string is a key without trying ULID parse — use ParseIDOrKey.
- Returning empty slices from GetIDs/GetKeys (breaks the nil-omit convention callers rely on).

<!-- archie:ai-end -->
