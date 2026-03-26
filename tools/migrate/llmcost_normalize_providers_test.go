package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLLMCostNormalizeProvidersMigration(t *testing.T) {
	// Rows that should be renamed
	nanogptRow := ulid.Make()    // nano-gpt → nanogpt
	nanoGptRow2 := ulid.Make()   // nano_gpt → nanogpt
	vertexRow := ulid.Make()     // vertex_ai-ext → vertex_ai
	xaiRow := ulid.Make()        // x-ai → xai
	azureRow := ulid.Make()      // azure_ai → azure
	bedrockRow := ulid.Make()    // bedrock_converse → bedrock
	geminiRow := ulid.Make()     // gemini → google
	prefixedModel := ulid.Make() // model_name has provider prefix

	// Canonical row that should block an alias rename (dedup scenario)
	canonicalNanogpt := ulid.Make() // nanogpt (already canonical)

	// Row that should not be touched
	openaiRow := ulid.Make()

	runner{
		stops: stops{
			{
				// Insert test data before the normalization migration.
				// Before: 20260326000000_llmcost_normalize_providers.up.sql
				version:   20260325154250,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Insert rows with various alias providers
					_, err := db.Exec(`
						INSERT INTO llm_cost_prices (
							id, created_at, updated_at,
							provider, model_id, model_name,
							input_per_token, output_per_token,
							cache_read_per_token, reasoning_per_token, cache_write_per_token,
							currency, source, effective_from
						) VALUES
							-- nano-gpt alias (should be renamed to nanogpt)
							($1, NOW(), NOW(), 'nano-gpt', 'model-a', 'Model A',
							 0.001, 0.002, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- nano_gpt alias, same model (alias-vs-alias collision, one should be soft-deleted)
							($2, NOW(), NOW(), 'nano_gpt', 'model-a', 'Model A',
							 0.001, 0.002, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- canonical nanogpt, same model (alias→canonical conflict, aliases should be soft-deleted)
							($3, NOW(), NOW(), 'nanogpt', 'model-a', 'Model A',
							 0.001, 0.002, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- vertex_ai-ext → vertex_ai
							($4, NOW(), NOW(), 'vertex_ai-ext', 'gemini-pro', 'Gemini Pro',
							 0.001, 0.002, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- x-ai → xai
							($5, NOW(), NOW(), 'x-ai', 'grok-1', 'Grok 1',
							 0.005, 0.01, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- azure_ai → azure
							($6, NOW(), NOW(), 'azure_ai', 'gpt-4', 'GPT 4',
							 0.03, 0.06, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- bedrock_converse → bedrock
							($7, NOW(), NOW(), 'bedrock_converse', 'claude-3', 'Claude 3',
							 0.008, 0.024, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- gemini → google
							($8, NOW(), NOW(), 'gemini', 'gemini-1.5', 'Gemini 1.5',
							 0.001, 0.002, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- model with provider prefix in model_name (should be stripped)
							($9, NOW(), NOW(), 'openai', 'gpt-3.5-turbo', 'azure/gpt-3.5-turbo',
							 0.0005, 0.0015, 0, 0, 0, 'USD', 'test', '2026-01-01'),
							-- openai row that should not be touched
							($10, NOW(), NOW(), 'openai', 'gpt-4o', 'GPT-4o',
							 0.005, 0.015, 0, 0, 0, 'USD', 'test', '2026-01-01')
					`,
						nanogptRow.String(),
						nanoGptRow2.String(),
						canonicalNanogpt.String(),
						vertexRow.String(),
						xaiRow.String(),
						azureRow.String(),
						bedrockRow.String(),
						geminiRow.String(),
						prefixedModel.String(),
						openaiRow.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// Verify normalization results after the migration.
				// After: 20260326000000_llmcost_normalize_providers.up.sql
				version:   20260326000000,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var provider, modelName string
					var deletedAt sql.NullTime

					// nano-gpt should be soft-deleted (canonical nanogpt exists)
					err := db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, nanogptRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.True(t, deletedAt.Valid, "nano-gpt row should be soft-deleted because canonical nanogpt exists")

					// nano_gpt should also be soft-deleted
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, nanoGptRow2.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.True(t, deletedAt.Valid, "nano_gpt row should be soft-deleted because canonical nanogpt exists")

					// canonical nanogpt should be untouched
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, canonicalNanogpt.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "nanogpt", provider)
					require.False(t, deletedAt.Valid, "canonical nanogpt should not be soft-deleted")

					// vertex_ai-ext → vertex_ai
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, vertexRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "vertex_ai", provider)
					require.False(t, deletedAt.Valid)

					// x-ai → xai
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, xaiRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "xai", provider)
					require.False(t, deletedAt.Valid)

					// azure_ai → azure
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, azureRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "azure", provider)
					require.False(t, deletedAt.Valid)

					// bedrock_converse → bedrock
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, bedrockRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "bedrock", provider)
					require.False(t, deletedAt.Valid)

					// gemini → google
					err = db.QueryRow(`SELECT provider, deleted_at FROM llm_cost_prices WHERE id = $1`, geminiRow.String()).Scan(&provider, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "google", provider)
					require.False(t, deletedAt.Valid)

					// model_name prefix stripped: "azure/gpt-3.5-turbo" → "gpt-3.5-turbo"
					err = db.QueryRow(`SELECT model_name FROM llm_cost_prices WHERE id = $1`, prefixedModel.String()).Scan(&modelName)
					require.NoError(t, err)
					require.Equal(t, "gpt-3.5-turbo", modelName, "provider prefix should be stripped from model_name")

					// openai row should be untouched
					err = db.QueryRow(`SELECT provider, model_name, deleted_at FROM llm_cost_prices WHERE id = $1`, openaiRow.String()).Scan(&provider, &modelName, &deletedAt)
					require.NoError(t, err)
					require.Equal(t, "openai", provider)
					require.Equal(t, "GPT-4o", modelName, "model_name without prefix should be untouched")
					require.False(t, deletedAt.Valid)

					// Verify total active row count
					var activeCount int
					err = db.QueryRow(`SELECT COUNT(*) FROM llm_cost_prices WHERE deleted_at IS NULL`).Scan(&activeCount)
					require.NoError(t, err)
					require.Equal(t, 8, activeCount, "should have 8 active rows (10 original minus 2 soft-deleted nanogpt aliases)")
				},
			},
		},
	}.Test(t)
}
