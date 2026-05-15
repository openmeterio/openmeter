# entity

<!-- archie:ai-start -->

> Pure domain types for the secret package: SecretID (namespaced composite key encoding AppID+ID+Key), Secret (ID + value), all input structs with Validate() methods, and the SecretNotFoundError sentinel. Contains zero persistence logic and zero Ent imports — intentional to break the import cycle between openmeter/app and openmeter/secret.

## Patterns

**Validate() on every input type** — All input structs (CreateAppSecretInput, UpdateAppSecretInput) expose Validate() error. The service layer calls Validate() unconditionally before delegating to the adapter. Validation errors are wrapped in models.NewGenericValidationError. (`func (i CreateAppSecretInput) Validate() error {
    if err := i.AppID.Validate(); err != nil {
        return models.NewGenericValidationError(errors.New("app id is invalid"))
    }
    if i.Key == "" { return models.NewGenericValidationError(errors.New("key is required")) }
    return nil
}`)
**Embed models.NamespacedID in identity types** — SecretID embeds models.NamespacedID so it carries both Namespace and ID and can chain NamespacedID.Validate(). Validate() chains: NamespacedID.Validate() → AppID.Validate() → Key check. (`type SecretID struct {
    models.NamespacedID
    AppID app.AppID
    Key   string
}`)
**Type aliases for symmetric input types** — GetAppSecretInput and DeleteAppSecretInput are type aliases for SecretID — no redundant wrapper structs when the input is just an identifier. (`type GetAppSecretInput = SecretID
type DeleteAppSecretInput = SecretID`)
**Error wraps models.GenericNotFoundError with compile-time assertion** — SecretNotFoundError wraps models.NewGenericNotFoundError so GenericErrorEncoder maps it to 404 automatically. `var _ models.GenericError = (*SecretNotFoundError)(nil)` asserts interface compliance. (`var _ models.GenericError = (*SecretNotFoundError)(nil)
func NewSecretNotFoundError(id SecretID) *SecretNotFoundError {
    return &SecretNotFoundError{err: models.NewGenericNotFoundError(fmt.Errorf("app with id %s not found", id.ID))}
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | SecretID and Secret domain types plus NewSecretID constructor and all Validate() methods. | SecretID.Validate() chains NamespacedID.Validate() → AppID.Validate() → Key check — all three must pass. Skipping AppID.Validate() allows invalid IDs to reach the adapter. |
| `input.go` | All four input structs. GetAppSecretInput and DeleteAppSecretInput are type aliases for SecretID, not new types. | UpdateAppSecretInput validates both AppID and SecretID independently. New fields must be validated here, not duplicated in the service layer. |
| `errors.go` | SecretNotFoundError sentinel wrapping models.GenericNotFoundError, plus IsSecretNotFoundError helper using errors.As. | Always use models.NewGenericNotFoundError as the inner error so the HTTP encoder produces 404 without special-casing this type. |

## Anti-Patterns

- Adding persistence logic or importing openmeter/ent/db — this package must remain import-cycle-free and dependency-free
- Returning raw fmt.Errorf from Validate() instead of models.NewGenericValidationError — breaks HTTP status code mapping
- Creating new input structs without a Validate() method — service layer calls Validate() unconditionally
- Duplicating validation logic in the adapter that already exists in input Validate() methods
- Adding SecretID fields without extending Validate() to check them — silent invalid state reaches the adapter

## Decisions

- **Input types live in a separate entity/ sub-package rather than in the root secret package** — Breaks the import cycle: input.go imports openmeter/app for AppID, while openmeter/app imports openmeter/secret — placing shared types in secretentity lets both import it without a cycle.
- **GetAppSecretInput and DeleteAppSecretInput are type aliases (=) not new types** — When the input is just an identifier, a wrapper struct adds no value and creates a translation cost; aliases allow callers to pass a SecretID directly.

<!-- archie:ai-end -->
