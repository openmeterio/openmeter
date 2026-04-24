# entity

<!-- archie:ai-start -->

> Pure domain types for the secret package: SecretID (namespaced composite key), Secret (ID + value), all input structs, and the SecretNotFoundError sentinel. No persistence logic lives here.

## Patterns

**Validate() on every input type** — All input structs (CreateAppSecretInput, UpdateAppSecretInput) expose a Validate() error method that wraps field errors in models.NewGenericValidationError. Service layer calls Validate() before delegating to the adapter. (`func (i CreateAppSecretInput) Validate() error { if i.Key == "" { return models.NewGenericValidationError(errors.New("key is required")) }; return nil }`)
**Embed models.NamespacedID in identity types** — SecretID embeds models.NamespacedID so it carries both Namespace and ID and can reuse NamespacedID.Validate(). (`type SecretID struct { models.NamespacedID; AppID app.AppID; Key string }`)
**Error wraps models.GenericNotFoundError** — SecretNotFoundError wraps models.NewGenericNotFoundError so the HTTP encoder maps it to 404 automatically. Add `var _ models.GenericError = (*SecretNotFoundError)(nil)` for compile-time verification. (`var _ models.GenericError = (*SecretNotFoundError)(nil)`)
**Type aliases for symmetric input types** — GetAppSecretInput and DeleteAppSecretInput are type aliases for SecretID, avoiding redundant wrapper structs when the input is just an identifier. (`type GetAppSecretInput = SecretID`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | SecretID and Secret domain types plus NewSecretID constructor and Validate methods. | SecretID.Validate() chains NamespacedID.Validate() → AppID.Validate() → Key check; all must pass. Skipping AppID.Validate() lets invalid IDs reach the adapter. |
| `input.go` | All four input structs. GetAppSecretInput and DeleteAppSecretInput are aliases, not new types. | UpdateAppSecretInput has both AppID and SecretID — Validate() calls both. New fields must be validated here, not in the service. |
| `errors.go` | SecretNotFoundError sentinel and IsSecretNotFoundError helper. | Always use models.NewGenericNotFoundError as the inner error so the HTTP layer maps it to 404 without special-casing. |

## Anti-Patterns

- Adding persistence logic or Ent imports to this package — it must remain a pure domain type package
- Returning raw errors instead of models.NewGenericValidationError from Validate() — breaks HTTP status code mapping
- Creating new input structs without a Validate() method — service layer calls Validate() unconditionally
- Duplicating validation logic in the adapter that already exists in input Validate() methods

## Decisions

- **Input types live in a separate entity sub-package rather than in the root secret package** — Breaks the import cycle: input.go imports openmeter/app for AppID, while openmeter/app imports openmeter/secret — placing types in a sub-package (secretentity) lets both import it without a cycle.

<!-- archie:ai-end -->
