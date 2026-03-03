package sync

import (
	"regexp"
	"strings"
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

// versionSuffix matches date-based version suffixes like -20241022, -20250219
var versionSuffix = regexp.MustCompile(`-\d{8}$`)

func (n *defaultNormalizer) Normalize(rawID string, provider string) (string, string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelID := strings.ToLower(strings.TrimSpace(rawID))

	// Strip date version suffixes (e.g., -20241022)
	modelID = versionSuffix.ReplaceAllString(modelID, "")

	// Normalize common provider names
	provider = normalizeProvider(provider)

	return provider, modelID
}

func normalizeProvider(provider string) string {
	switch provider {
	case "openai", "azure", "azure_ai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "google", "vertex_ai", "gemini":
		return "google"
	case "amazon", "aws", "bedrock":
		return "amazon"
	case "meta", "facebook":
		return "meta"
	case "deepseek":
		return "deepseek"
	case "mistral", "mistralai":
		return "mistral"
	case "cohere":
		return "cohere"
	case "xai":
		return "xai"
	case "minimax":
		return "minimax"
	default:
		return provider
	}
}
