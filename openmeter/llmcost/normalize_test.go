package llmcost

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeModelIDProviderNames(t *testing.T) {
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
			provider, _ := NormalizeModelID("some-model", tt.input)
			assert.Equal(t, tt.expected, provider)
		})
	}
}

func TestNormalizeModelIDCaseAndWhitespace(t *testing.T) {
	t.Run("lowercases provider", func(t *testing.T) {
		provider, _ := NormalizeModelID("model", "OpenAI")
		assert.Equal(t, "openai", provider)
	})

	t.Run("lowercases model ID", func(t *testing.T) {
		_, modelID := NormalizeModelID("GPT-4o", "openai")
		assert.Equal(t, "gpt-4o", modelID)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		provider, modelID := NormalizeModelID("  gpt-4  ", "  openai  ")
		assert.Equal(t, "openai", provider)
		assert.Equal(t, "gpt-4", modelID)
	})
}

func TestNormalizeModelIDVersionSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips 8-digit date suffix", "claude-3-5-sonnet-20241022", "claude-3-5-sonnet"},
		{"strips another 8-digit date suffix", "gpt-4-turbo-20240409", "gpt-4-turbo"},
		{"strips hyphenated date suffix", "gpt-5-nano-2025-08-07", "gpt-5-nano"},
		{"strips hyphenated date suffix 2", "gpt-4o-2024-08-06", "gpt-4o"},
		{"no suffix unchanged", "gpt-4o", "gpt-4o"},
		{"suffix in middle unchanged", "model-20241022-beta", "model-20241022-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := NormalizeModelID(tt.input, "openai")
			assert.Equal(t, tt.expected, modelID)
		})
	}
}
