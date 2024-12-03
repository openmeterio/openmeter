package hasher

import "github.com/cespare/xxhash/v2"

func NewHash(data []byte) Hash {
	return xxhash.Sum64(data)
}
