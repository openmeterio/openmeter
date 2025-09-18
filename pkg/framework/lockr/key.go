package lockr

import (
	"errors"
	"fmt"
	"strings"

	xxhash "github.com/zeebo/xxh3"
)

var stringSeparator = ":"

// Key is a unique identifier for a resource and a scope that can be locked
type Key interface {
	String() string
	Hash64() uint64
}

// NewKey constructs a key for the given scopes.
// Scope is always required and must be non-empty.
func NewKey(scopes ...string) (Key, error) {
	if len(scopes) == 0 {
		return nil, errors.New("at least one scope is required")
	}

	for idx, s := range scopes {
		if s == "" {
			return nil, fmt.Errorf("scope cannot be empty [index=%d]", idx)
		}

		if strings.Contains(s, stringSeparator) {
			return nil, fmt.Errorf("scope cannot contain %q [index=%d]", stringSeparator, idx)
		}
	}

	return &key{scopes: scopes}, nil
}

type key struct {
	scopes []string
}

var _ Key = (*key)(nil)

func (k *key) String() string {
	return strings.Join(k.scopes, stringSeparator)
}

// Hash64 translates the key string to a 64bit keyspace via hashing it
func (k *key) Hash64() uint64 {
	h := xxhash.New()

	_, _ = h.WriteString(k.String())

	return h.Sum64()
}
