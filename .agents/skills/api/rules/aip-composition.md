# Composition over inheritance

The `composition-over-inheritance` linter rule (warning) flags `extends` on a base without `@discriminator`. Prefer:

- **Spread** (`...BaseModel`) for composing fields into a new model.
- **`model Foo is Bar`** for aliasing or narrowing a template instantiation.
- **`@discriminator`** on the base model only when a true polymorphic union is needed.
