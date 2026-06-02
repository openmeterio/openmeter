# OpenMeter error response types

OpenMeter wires AIP-193 (`rules/aip-193-errors.md`) onto operations through the `Common.*` types in `api/spec/packages/aip/src/common/`. Use `Common.ErrorResponses` (= `BadRequest | Unauthorized | Forbidden`) on every operation, then add specific types explicitly. Types with ★ are OpenMeter extensions beyond what AIP-193 documents by name.

| Type                          | Status | Source      | When to add explicitly                              |
| ----------------------------- | ------ | ----------- | --------------------------------------------------- |
| `Common.NotFound`             | 404    | AIP-193     | GET, PATCH, PUT, DELETE by ID                       |
| `Common.Conflict`             | 409    | AIP-193     | Create operations that may conflict                 |
| `Common.Gone`                 | 410    | ★ OpenMeter | PUT/PATCH when the resource was soft-deleted        |
| `Common.PayloadTooLarge`      | 413    | ★ OpenMeter | Endpoints accepting large bodies or bulk operations |
| `Common.UnprocessableContent` | 422    | ★ OpenMeter | Semantically invalid requests                       |

```tsp
get(@path meterId: Shared.ULID): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;
```

For inline errors returned **inside** a 2xx response body (partial successes, pre-flight validation on draft resources), see `rules/inline-errors.md`.
