package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizerProviderNames(t *testing.T) {
	n := NewDefaultNormalizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"openai", "openai"},
		{"azure", "openai"},
		{"azure_ai", "openai"},
		{"anthropic", "anthropic"},
		{"google", "google"},
		{"vertex_ai", "google"},
		{"gemini", "google"},
		{"amazon", "amazon"},
		{"aws", "amazon"},
		{"bedrock", "amazon"},
		{"meta", "meta"},
		{"facebook", "meta"},
		{"deepseek", "deepseek"},
		{"mistral", "mistral"},
		{"mistralai", "mistral"},
		{"cohere", "cohere"},
		{"xai", "xai"},
		{"minimax", "minimax"},
		{"unknown_provider", "unknown_provider"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			provider, _ := n.Normalize("some-model", tt.input)
			assert.Equal(t, tt.expected, provider)
		})
	}
}

func TestNormalizerCaseAndWhitespace(t *testing.T) {
	n := NewDefaultNormalizer()

	t.Run("lowercases provider", func(t *testing.T) {
		provider, _ := n.Normalize("model", "OpenAI")
		assert.Equal(t, "openai", provider)
	})

	t.Run("lowercases model ID", func(t *testing.T) {
		_, modelID := n.Normalize("GPT-4o", "openai")
		assert.Equal(t, "gpt-4o", modelID)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		provider, modelID := n.Normalize("  gpt-4  ", "  openai  ")
		assert.Equal(t, "openai", provider)
		assert.Equal(t, "gpt-4", modelID)
	})
}

func TestNormalizerVersionSuffix(t *testing.T) {
	n := NewDefaultNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips date suffix", "gpt-4o-2024-08-06", "gpt-4o-2024-08-06"},                  // Not 8-digit, kept
		{"strips 8-digit date suffix", "claude-3-5-sonnet-20241022", "claude-3-5-sonnet"}, // 8-digit, stripped
		{"strips another date suffix", "gpt-4-turbo-20240409", "gpt-4-turbo"},             // 8-digit, stripped
		{"no suffix unchanged", "gpt-4o", "gpt-4o"},
		{"suffix in middle unchanged", "model-20241022-beta", "model-20241022-beta"}, // Not at end
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := n.Normalize(tt.input, "openai")
			assert.Equal(t, tt.expected, modelID)
		})
	}
}
