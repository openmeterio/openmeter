# Visibility

Use `@visibility` on every field to control which operations expose it. Do not rely on defaults.

- `Lifecycle.Read` — returned by any operation (GET, list, create/update response, etc.)
- `Lifecycle.Create` — accepted in create request bodies (POST)
- `Lifecycle.Update` — accepted in update request bodies (PATCH)

Server-managed fields (`id`, timestamps) must be `Lifecycle.Read` only.
