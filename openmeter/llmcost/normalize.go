package llmcost

import (
	"regexp"
	"strings"
)

// versionSuffix matches date-based version suffixes:
//   - 8-digit compact: -20241022, -20250219
//   - hyphenated:      -2025-08-07, -2024-08-06
var versionSuffix = regexp.MustCompile(`(-\d{8}|-\d{4}-\d{2}-\d{2})$`)

// NormalizeModelID maps a raw model ID and provider to their canonical forms.
// It lowercases, trims whitespace, strips date version suffixes, and normalizes
// provider aliases (e.g. "azure" → "openai").
func NormalizeModelID(rawID string, provider string) (canonicalProvider string, canonicalModelID string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelID := strings.ToLower(strings.TrimSpace(rawID))

	// Strip date version suffixes (e.g., -20241022 or -2025-08-07)
	modelID = versionSuffix.ReplaceAllString(modelID, "")

	// Normalize common provider names
	provider = NormalizeProvider(provider)

	return provider, modelID
}

// NormalizeProvider maps alternative provider names to their canonical form.
func NormalizeProvider(provider string) string {
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
