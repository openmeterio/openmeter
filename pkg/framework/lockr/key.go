package lockr

import (
	"errors"
	"strings"

	"github.com/cespare/xxhash/v2"
)

var stringSeparator = ":"

// Key is a unique identifier for a resource and a scope that can be locked
type Key interface {
	String() string
	Hash64() uint64
}

// NewKey constructs a key for the given scopes.
// Scope is always required and must be non-empty.
func NewKey(scope string, scopes ...string) (Key, error) {
	joined := []string{scope}
	joined = append(joined, scopes...)

	for _, s := range joined {
		if s == "" {
			return nil, errors.New("scope cannot be empty")
		}

		if strings.Contains(s, stringSeparator) {
			return nil, errors.New("scope cannot contain ':'")
		}
	}

	return &key{scopes: joined}, nil
}

type key struct {
	scopes []string
}

var _ Key = (*key)(nil)

func (k *key) String() string {
	return strings.Join(k.scopes, stringSeparator)
}

// KeySpaceEntry translates the key string to a 64bit keyspace via hashing it
func (k *key) Hash64() uint64 {
	h := xxhash.New()

	for _, s := range k.scopes {
		_, _ = h.WriteString(s)
	}

	return h.Sum64()
}
