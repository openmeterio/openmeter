# entity

<!-- archie:ai-start -->

> Domain types and validation rules for the progressmanager package: ProgressID, Progress, and the GetProgressInput/UpsertProgressInput method-input wrappers. Pure data + Validate(), no I/O.

## Patterns

**errors.Join validation accumulation** — Every Validate() collects into `var errs []error` and returns errors.Join(errs...) — it never returns on the first failure. Nested validators are wrapped with field context. (`if err := a.NamespacedModel.Validate(); err != nil { errs = append(errs, fmt.Errorf("namespaced model: %w", err)) }`)
**Embedded ID + NamespacedModel** — ProgressID embeds models.NamespacedModel (json inline) plus an ID string; Progress embeds ProgressID; inputs embed the domain type they carry. Reuse embedding rather than duplicating fields. (`type Progress struct { ProgressID `json:"id"`; Success uint64 ... }`)
**Counter invariants in Validate** — Progress.Validate enforces Success+Failed<=Total, zero counters when Total==0, and non-zero UpdatedAt. New counter fields must extend these checks. (`if a.Success+a.Failed > a.Total { errs = append(errs, errors.New("success and failed must be less than or equal to total")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `progressmanager.go` | Defines ProgressID, Progress, GetProgressInput, UpsertProgressInput and their Validate() methods. | Receivers are named `a` despite the types not being adapters; UpdatedAt is required (IsZero rejected). Input wrappers exist purely to keep method signatures stable as fields grow. |

## Anti-Patterns

- Returning on the first validation error instead of accumulating with errors.Join.
- Adding counter fields without updating the Success+Failed<=Total / Total==0 invariants.
- Importing anything beyond pkg/models here — this folder must stay I/O-free.

## Decisions

- **Method inputs are dedicated structs (GetProgressInput/UpsertProgressInput) wrapping the domain type.** — Keeps adapter/service signatures stable and gives each call its own Validate() seam.

## Example: Build and validate an upsert input

```
import (
  "time"
  "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
  "github.com/openmeterio/openmeter/pkg/models"
)

in := entity.UpsertProgressInput{Progress: entity.Progress{
  ProgressID: entity.ProgressID{NamespacedModel: models.NamespacedModel{Namespace: ns}, ID: id},
  Success: 3, Total: 10, UpdatedAt: time.Now(),
}}
if err := in.Validate(); err != nil { return err }
```

<!-- archie:ai-end -->
