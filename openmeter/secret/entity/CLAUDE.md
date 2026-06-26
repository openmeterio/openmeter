# entity

<!-- archie:ai-start -->

> Domain types for the secret package: SecretID (a NamespacedID scoped to an app + key), Secret (id + value), the input structs for each operation, and the typed SecretNotFoundError. This is the shared vocabulary imported by adapter, service, server, and the Stripe app.

## Patterns

**Self-validating value objects** — Every type (SecretID, Secret, CreateAppSecretInput, UpdateAppSecretInput) exposes a Validate() error that returns models.NewGenericValidationError wrapping the first failure. (`func (i CreateAppSecretInput) Validate() error { if err := i.AppID.Validate(); err != nil { return models.NewGenericValidationError(errors.New("app id is invalid")) } ... }`)
**NamespacedID embedding for tenancy** — SecretID embeds models.NamespacedID; NewSecretID derives Namespace from appID.Namespace so every secret is tenant-scoped. (`SecretID{ NamespacedID: models.NamespacedID{Namespace: appID.Namespace, ID: id}, AppID: appID, Key: key }`)
**Type aliases for read/delete inputs** — GetAppSecretInput and DeleteAppSecretInput are plain aliases for SecretID (= SecretID), not new structs, because those operations need nothing beyond the id. (`type GetAppSecretInput = SecretID; type DeleteAppSecretInput = SecretID`)
**Typed not-found error with detector** — SecretNotFoundError wraps models.NewGenericNotFoundError and ships an IsSecretNotFoundError(err) helper using errors.As; it asserts models.GenericError compliance. (`var _ models.GenericError = (*SecretNotFoundError)(nil); func IsSecretNotFoundError(err error) bool { var e *SecretNotFoundError; return errors.As(err, &e) }`)
**Package name differs from directory** — The package is named secretentity (not entity); importers alias it as secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity". (`package secretentity`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | Defines SecretID + NewSecretID constructor and Secret, each with Validate(). | Key is required by SecretID.Validate(); a SecretID is invalid without it even if NamespacedID is set. |
| `input.go` | Create/Update input structs plus the Get/Delete type aliases. | CreateAppSecretInput.Validate flattens AppID errors to a generic 'app id is invalid' message, while UpdateAppSecretInput.Validate returns the underlying AppID/SecretID error verbatim — they are deliberately inconsistent. |
| `errors.go` | SecretNotFoundError type and IsSecretNotFoundError detector. | Use IsSecretNotFoundError for detection rather than direct type asserts; the message string says 'app with id ... not found' even though it is a secret error. |

## Anti-Patterns

- Constructing SecretID literals directly instead of via NewSecretID — you risk leaving Namespace unset.
- Returning raw errors from Validate() instead of wrapping in models.NewGenericValidationError.
- Adding a value field to SecretID — the value lives only on Secret, never on the ID.

## Decisions

- **Get/Delete inputs are aliases of SecretID.** — Those operations are fully addressed by the id; a separate struct would add no fields.

## Example: Building and validating a namespaced secret id

```
import secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"

id := secretentity.NewSecretID(appID, value, "stripe-api-key")
if err := id.Validate(); err != nil { return err }
```

<!-- archie:ai-end -->
