package hasher

type Hash = uint64

type Hasher interface {
	Hash() Hash
}
