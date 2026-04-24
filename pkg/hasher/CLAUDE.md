# hasher

<!-- archie:ai-start -->

> Minimal hashing utility package: defines a Hash type alias (uint64), a Hasher interface, and a single xxhash-backed implementation. Used wherever a fast, non-cryptographic content hash is needed (e.g. lock key derivation in pkg/framework/lockr).

## Patterns

**Hash = uint64 type alias** — Hash is a type alias, not a named type — it is interchangeable with uint64 at the call site. Consumers that need nominal typing should define their own wrapper. (`var h hasher.Hash = hasher.NewHash([]byte("key"))`)
**Hasher interface for injectable hashing** — Structs that need to hash themselves implement Hasher by returning a Hash from Hash(). Inject via the interface, not via direct xxhash calls. (`type myKey struct{ ... }
func (k myKey) Hash() hasher.Hash { return hasher.NewHash([]byte(k.id)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hasher.go` | Defines Hash type and Hasher interface. Source of truth for the contract. | Hash is a type alias — changing it to a named type would be a breaking change for all callers assigning uint64 literals. |
| `xxhash.go` | NewHash([]byte) is the only concrete hash function. Uses cespare/xxhash v2. | xxhash is non-cryptographic — never use for security-sensitive hashing. |

## Anti-Patterns

- Using hasher.NewHash for cryptographic purposes (passwords, signatures) — use crypto/sha256 or similar instead
- Importing xxhash directly in domain packages instead of going through hasher.NewHash

<!-- archie:ai-end -->
