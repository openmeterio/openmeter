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
// provider aliases (e.g. "azure_ai" → "azure", "gemini" → "google").
func NormalizeModelID(provider string, modelID string) (canonicalProvider string, canonicalModelID string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelID = strings.ToLower(strings.TrimSpace(modelID))

	// Strip date version suffixes (e.g., -20241022 or -2025-08-07)
	modelID = versionSuffix.ReplaceAllString(modelID, "")

	// Normalize common provider names
	provider = NormalizeProvider(provider)

	return provider, modelID
}

// NormalizeProvider maps alternative provider names to their canonical form.
// It lowercases and trims whitespace before matching.
func NormalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))

	// Normalize vertex_ai sub-providers (e.g., "vertex_ai-language-models", "vertex_ai-text-models")
	if strings.HasPrefix(provider, "vertex_ai-") || strings.HasPrefix(provider, "vertex_ai_") {
		return "vertex_ai"
	}

	switch provider {
	// Hosting providers are kept separate from the model vendors they host,
	// because pricing can differ (e.g., Azure gpt-4 is 2x OpenAI gpt-4).
	case "openai":
		return "openai"
	case "azure", "azure_ai":
		return "azure"
	case "anthropic":
		return "anthropic"
	case "bedrock", "bedrock_converse":
		return "bedrock"
	case "google", "gemini":
		return "google"
	case "vertex_ai":
		return "vertex_ai"
	case "amazon", "aws":
		return "amazon"
	case "meta", "facebook":
		return "meta"
	case "deepseek":
		return "deepseek"
	case "mistral", "mistralai":
		return "mistral"
	case "cohere":
		return "cohere"
	case "xai", "x-ai":
		return "xai"
	case "minimax":
		return "minimax"
	case "nano-gpt", "nano_gpt", "nanogpt":
		return "nanogpt"
	default:
		return provider
	}
}
