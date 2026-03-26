package llmcost

import (
	"regexp"
	"strings"
)

// versionSuffix matches date-based version suffixes:
//   - 8-digit compact: -20241022, -20250219
//   - hyphenated:      -2025-08-07, -2024-08-06
var versionSuffix = regexp.MustCompile(`(-\d{8}|-\d{4}-\d{2}-\d{2})$`)

// bedrockVersionSuffix matches AWS Bedrock model version suffixes like -v1:0, -v2:0.
var bedrockVersionSuffix = regexp.MustCompile(`-v\d+:\d+$`)

// bedrockRegionPrefix matches AWS region prefixes on Bedrock model IDs (e.g., "eu.", "us.", "ap.").
var bedrockRegionPrefix = regexp.MustCompile(`^(us|eu|ap)\.`)

// dotVersion matches dots between digits in version numbers (e.g., "3.5" in "claude-3.5-sonnet").
// This normalizes "claude-3.5-sonnet" to "claude-3-5-sonnet" without touching namespace dots like "anthropic.claude".
var dotVersion = regexp.MustCompile(`(\d)\.(\d)`)

// modelAliases maps alternative model IDs to their canonical form.
// Applied after all other normalization steps.
var modelAliases = map[string]string{
	// DeepSeek API names vs marketing names
	"deepseek-chat":     "deepseek-v3",
	"deepseek-reasoner": "deepseek-r1",
}

// NormalizeModelID maps a raw model ID and provider to their canonical forms.
// It lowercases, trims whitespace, strips date version suffixes, and normalizes
// provider aliases (e.g. "azure_ai" → "azure", "gemini" → "google").
func NormalizeModelID(provider string, modelID string) (canonicalProvider string, canonicalModelID string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelID = strings.ToLower(strings.TrimSpace(modelID))

	// Strip AWS Bedrock version suffixes first (e.g., -v1:0, -v2:0) so that
	// date suffixes that precede them become terminal and can be stripped next.
	modelID = bedrockVersionSuffix.ReplaceAllString(modelID, "")

	// Strip date version suffixes (e.g., -20241022 or -2025-08-07)
	modelID = versionSuffix.ReplaceAllString(modelID, "")

	// Strip AWS Bedrock region prefixes (e.g., "eu.", "us.", "ap.")
	modelID = bedrockRegionPrefix.ReplaceAllString(modelID, "")

	// Normalize dots between digits to hyphens (e.g., "claude-3.5-sonnet" → "claude-3-5-sonnet")
	modelID = dotVersion.ReplaceAllString(modelID, "${1}-${2}")

	// Apply model-specific aliases
	if canonical, ok := modelAliases[modelID]; ok {
		modelID = canonical
	}

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
	case "bedrock", "bedrock_converse", "amazon-bedrock":
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
