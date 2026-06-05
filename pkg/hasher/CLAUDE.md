# hasher

<!-- archie:ai-start -->

> Tiny utility defining a stable hashing contract: a Hash type alias and Hasher interface, with an xxhash-backed implementation. Used to derive content hashes for domain objects (e.g. productcatalog).

## Patterns

**Hash is a uint64 type alias** — Hash is declared as `type Hash = uint64` (alias, not a named type), so it is interchangeable with raw uint64 at call sites. (`type Hash = uint64`)
**Hasher interface returns Hash** — Types that participate in hashing implement `Hasher` with a single `Hash() Hash` method; consumers depend on the interface, not the concrete algorithm. (`type Hasher interface { Hash() Hash }`)
**Single hashing primitive via xxhash** — All raw-byte hashing goes through `NewHash([]byte) Hash` which delegates to `xxhash.Sum64`. Do not introduce alternative hash algorithms here. (`func NewHash(data []byte) Hash { return xxhash.Sum64(data) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hasher.go` | Declares the Hash alias and Hasher interface — the public contract. | Hash is an alias; changing it to a named type would break implicit uint64 conversions across importers. |
| `xxhash.go` | The only concrete hashing function, NewHash, wrapping cespare/xxhash/v2. | xxhash is non-cryptographic; never use these hashes for security/signing purposes. |

## Anti-Patterns

- Adding crypto hashing (sha256/md5) or a second algorithm in this package instead of keeping a single xxhash primitive.
- Redefining Hash as a distinct named type, breaking uint64 interchangeability for existing callers.

## Decisions

- **xxhash chosen as the sole hashing primitive.** — Fast, non-cryptographic 64-bit hashing is sufficient for content fingerprinting/cache keys in productcatalog.

## Example: Implement Hasher for a domain object

```
import "github.com/openmeterio/openmeter/pkg/hasher"

func (o Object) Hash() hasher.Hash {
	return hasher.NewHash([]byte(o.ID + o.Name))
}
```

<!-- archie:ai-end -->
