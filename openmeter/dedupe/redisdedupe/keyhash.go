package redisdedupe

import (
	"crypto/md5"
	"encoding/base64"
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
// md5 is deprecated, but it's still a good hash function for our use case and it's just 128 bits, we should
// not care about specially crafted inputs that could lead to collisions.
// Raw size: 16 bytes, base64 size: ~22 chars (57% memory improvement in redis)
//
// Sha224 is 28 binary characters
// Raw size: 28 bytes, base64 size: ~40 chars (48% memory improvement)

func getKeyHash(itemKey string) string {
	hash := md5.Sum([]byte(itemKey))
	b64 := base64.StdEncoding.EncodeToString(hash[:])
	return b64
}
