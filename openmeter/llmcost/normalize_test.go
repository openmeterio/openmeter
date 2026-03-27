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
		{"azure", "azure"},
		{"azure_ai", "azure"},
		{"anthropic", "anthropic"},
		{"google", "google"},
		{"vertex_ai", "vertex_ai"},
		{"gemini", "google"},
		{"amazon", "amazon"},
		{"aws", "amazon"},
		{"bedrock", "bedrock"},
		{"bedrock_converse", "bedrock"},
		{"amazon-bedrock", "bedrock"},
		{"meta", "meta"},
		{"facebook", "meta"},
		{"deepseek", "deepseek"},
		{"mistral", "mistral"},
		{"mistralai", "mistral"},
		{"cohere", "cohere"},
		{"xai", "xai"},
		{"x-ai", "xai"},
		{"minimax", "minimax"},
		{"nano-gpt", "nanogpt"},
		{"nano_gpt", "nanogpt"},
		{"nanogpt", "nanogpt"},
		{"vertex_ai-language-models", "vertex_ai"},
		{"vertex_ai-text-models", "vertex_ai"},
		{"vertex_ai-chat-models", "vertex_ai"},
		{"vertex_ai_something", "vertex_ai"},
		{"unknown_provider", "unknown_provider"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			provider, _ := NormalizeModelID(tt.input, "some-model")
			assert.Equal(t, tt.expected, provider)
		})
	}
}

func TestNormalizeModelIDCaseAndWhitespace(t *testing.T) {
	t.Run("lowercases provider", func(t *testing.T) {
		provider, _ := NormalizeModelID("OpenAI", "model")
		assert.Equal(t, "openai", provider)
	})

	t.Run("lowercases model ID", func(t *testing.T) {
		_, modelID := NormalizeModelID("openai", "GPT-4o")
		assert.Equal(t, "gpt-4o", modelID)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		provider, modelID := NormalizeModelID("  openai  ", "  gpt-4  ")
		assert.Equal(t, "openai", provider)
		assert.Equal(t, "gpt-4", modelID)
	})
}

func TestNormalizeProviderCaseAndWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercases", "Azure", "azure"},
		{"trims whitespace", "  azure  ", "azure"},
		{"lowercases and trims", "  OpenAI  ", "openai"},
		{"mixed case unknown", "  MyProvider  ", "myprovider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeProvider(tt.input))
		})
	}
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
			_, modelID := NormalizeModelID("openai", tt.input)
			assert.Equal(t, tt.expected, modelID)
		})
	}
}

func TestNormalizeModelIDBedrockVersionSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips -v1:0", "anthropic.claude-3-5-sonnet-v1:0", "anthropic.claude-3-5-sonnet"},
		{"strips -v2:0", "anthropic.claude-3-5-sonnet-v2:0", "anthropic.claude-3-5-sonnet"},
		{"strips date then version", "anthropic.claude-3-5-sonnet-20240620-v1:0", "anthropic.claude-3-5-sonnet"},
		{"strips from nova model", "amazon.nova-pro-v1:0", "amazon.nova-pro"},
		{"strips from llama model", "meta.llama3-1-70b-instruct-v1:0", "meta.llama3-1-70b-instruct"},
		{"no version suffix unchanged", "anthropic.claude-sonnet-4-6", "anthropic.claude-sonnet-4-6"},
		{"non-bedrock model unchanged", "gpt-4o", "gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := NormalizeModelID("bedrock", tt.input)
			assert.Equal(t, tt.expected, modelID)
		})
	}
}

func TestNormalizeModelIDBedrockRegionPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips eu prefix", "eu.anthropic.claude-sonnet-4-6", "anthropic.claude-sonnet-4-6"},
		{"strips us prefix", "us.anthropic.claude-opus-4", "anthropic.claude-opus-4"},
		{"strips ap prefix", "ap.meta.llama3-1-70b-instruct", "meta.llama3-1-70b-instruct"},
		{"no prefix unchanged", "anthropic.claude-sonnet-4-6", "anthropic.claude-sonnet-4-6"},
		{"strips prefix and version", "us.anthropic.claude-3-5-sonnet-20241022-v1:0", "anthropic.claude-3-5-sonnet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := NormalizeModelID("bedrock", tt.input)
			assert.Equal(t, tt.expected, modelID)
		})
	}
}

func TestNormalizeModelIDDotVersion(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		input    string
		expected string
	}{
		{"claude 3.5 dot to hyphen", "anthropic", "claude-3.5-sonnet", "claude-3-5-sonnet"},
		{"claude 3.5 haiku", "anthropic", "claude-3.5-haiku", "claude-3-5-haiku"},
		{"gemini 2.0", "google", "gemini-2.0-flash", "gemini-2-0-flash"},
		{"already hyphens unchanged", "anthropic", "claude-3-5-sonnet", "claude-3-5-sonnet"},
		{"namespace dot preserved", "bedrock", "anthropic.claude-3-5-sonnet", "anthropic.claude-3-5-sonnet"},
		{"no version dots unchanged", "openai", "gpt-4o", "gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := NormalizeModelID(tt.provider, tt.input)
			assert.Equal(t, tt.expected, modelID)
		})
	}
}

func TestNormalizeModelIDAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"deepseek-chat to deepseek-v3", "deepseek-chat", "deepseek-v3"},
		{"deepseek-reasoner to deepseek-r1", "deepseek-reasoner", "deepseek-r1"},
		{"deepseek-v3 unchanged", "deepseek-v3", "deepseek-v3"},
		{"deepseek-r1 unchanged", "deepseek-r1", "deepseek-r1"},
		{"non-alias unchanged", "gpt-4o", "gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, modelID := NormalizeModelID("deepseek", tt.input)
			assert.Equal(t, tt.expected, modelID)
		})
	}
}
