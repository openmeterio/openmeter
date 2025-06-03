package redisdedupe

import (
	"encoding/base64"

	"github.com/zeebo/xxh3"
)

// Existing key format:
// orgId-source-id
//
// avg key size based on actual data is 77 characters

// New key format:
// base64(hash(orgId-source-id))
//
// Base64 introduces 33-37% overhead, but we don't have to fiddle with binary keys in the lua script.

// Hashes:
// xxh3 is a good hash function for our use case and it's just 128 bits (it's not cryptographic, but has decent
// collision resistance).
// Raw size: 16 bytes, base64 size: ~22 chars (57% memory improvement in redis)
//
// Sha224 is 28 binary characters
// Raw size: 28 bytes, base64 size: ~40 chars (48% memory improvement)
//
// Keyspace calculations:
// Assuming 10M events per day, we get 300M (3e8) events.
// The keyspace is 128 bits, which is 2^128 = 3.4e38 possible values.
//
// The probability of a cullusion is 3e8/3.4e38 ~ 1e-30, which is extremely low.

func GetKeyHash(itemKey string) string {
	hashBytes := xxh3.HashString128(itemKey).Bytes()
	b64 := base64.RawURLEncoding.EncodeToString(hashBytes[:])
	return b64
}
