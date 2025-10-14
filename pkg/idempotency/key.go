package idempotency

import (
	"fmt"

	"github.com/google/uuid"
)

// Key generates a new idempotency key in UUIDv7 format using random seed.
func Key() (string, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	return u.String(), nil
}

// MustKey is a helper function that panics if the key generation fails.
func MustKey() string {
	k, err := Key()
	if err != nil {
		panic(err)
	}

	return k
}
