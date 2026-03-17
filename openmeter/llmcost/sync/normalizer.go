package sync

import (
	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

// ModelIDNormalizer maps model identifiers to a canonical form.
// Source-specific normalization should be done by each Fetcher before
// returning prices; the normalizer only applies generic transforms.
type ModelIDNormalizer interface {
	Normalize(rawID string, provider string) (canonicalProvider string, canonicalModelID string)
}

type defaultNormalizer struct{}

func NewDefaultNormalizer() ModelIDNormalizer {
	return &defaultNormalizer{}
}

func (n *defaultNormalizer) Normalize(rawID string, provider string) (string, string) {
	return llmcost.NormalizeModelID(rawID, provider)
}
