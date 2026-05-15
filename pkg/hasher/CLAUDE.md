# hasher

<!-- archie:ai-start -->

> Minimal hashing utility: defines Hash (uint64 type alias) and Hasher interface, backed by a single xxhash implementation. Used for fast, non-cryptographic content hashing — primarily lock key derivation in pkg/framework/lockr.

## Patterns

**Hash = uint64 type alias** — Hash is a type alias (= uint64), not a named type — interchangeable with uint64 at call sites. Consumers needing nominal typing should define their own wrapper. (`var h hasher.Hash = hasher.NewHash([]byte("key"))`)
**Hasher interface for injectable hashing** — Structs that need to hash themselves implement Hasher by returning a Hash from Hash(). Inject via the interface, not via direct xxhash calls. (`type myKey struct{ id string }
func (k myKey) Hash() hasher.Hash { return hasher.NewHash([]byte(k.id)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hasher.go` | Defines Hash type alias and Hasher interface. Source of truth for the contract. | Changing Hash from a type alias to a named type would be a breaking change for all callers assigning uint64 literals. |
| `xxhash.go` | NewHash([]byte) Hash — the only concrete hash function, using cespare/xxhash v2. | xxhash is non-cryptographic — never use for security-sensitive hashing (passwords, signatures); use crypto/sha256 or similar instead. |

## Anti-Patterns

- Using hasher.NewHash for cryptographic purposes (authentication, signatures) — use crypto/sha256 or similar.
- Importing cespare/xxhash directly in domain packages instead of going through hasher.NewHash.
- Treating Hash as a named type in type assertions — it is a type alias for uint64.

## Decisions

- **Hash is a type alias rather than a named type.** — Allows callers (particularly lockr) to pass the result directly as uint64 to pg_advisory_xact_lock without an explicit conversion, reducing boilerplate.

<!-- archie:ai-end -->
