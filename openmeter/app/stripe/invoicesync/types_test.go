package invoicesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIdempotencyKey(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		key1 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		assert.Equal(t, key1, key2, "same inputs should produce same key")
	})

	t.Run("different invoice IDs produce different keys", func(t *testing.T) {
		key1 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv-2", "sess-1", 0, OpTypeInvoiceCreate)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("colon boundaries cannot alias different field splits", func(t *testing.T) {
		// Old colon-delimited concatenation mapped these to the same string; framing must keep them distinct.
		key1 := GenerateIdempotencyKey("inv:a", "sess", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv", "a:sess", 0, OpTypeInvoiceCreate)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different session IDs produce different keys", func(t *testing.T) {
		key1 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv-1", "sess-2", 0, OpTypeInvoiceCreate)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different sequences produce different keys", func(t *testing.T) {
		key1 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv-1", "sess-1", 1, OpTypeInvoiceCreate)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different op types produce different keys", func(t *testing.T) {
		key1 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		key2 := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceUpdate)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("key is hex encoded sha256", func(t *testing.T) {
		key := GenerateIdempotencyKey("inv-1", "sess-1", 0, OpTypeInvoiceCreate)
		require.Len(t, key, 64, "sha256 hex encoding should be 64 chars")
		assert.Regexp(t, `^[0-9a-f]{64}$`, key, "key should be lowercase hex")
	})
}

func TestEnumValues(t *testing.T) {
	t.Run("SyncPlanPhase values", func(t *testing.T) {
		values := SyncPlanPhaseDraft.Values()
		assert.Contains(t, values, "draft")
		assert.Contains(t, values, "issuing")
		assert.Contains(t, values, "delete")
		assert.Len(t, values, 3)
	})

	t.Run("OpType values", func(t *testing.T) {
		values := OpTypeInvoiceCreate.Values()
		assert.Contains(t, values, "invoice_create")
		assert.Contains(t, values, "invoice_update")
		assert.Contains(t, values, "invoice_delete")
		assert.Contains(t, values, "invoice_finalize")
		assert.Contains(t, values, "line_item_add")
		assert.Contains(t, values, "line_item_update")
		assert.Contains(t, values, "line_item_remove")
		assert.Len(t, values, 7)
	})

	t.Run("OpStatus values", func(t *testing.T) {
		values := OpStatusPending.Values()
		assert.Contains(t, values, "pending")
		assert.Contains(t, values, "completed")
		assert.Contains(t, values, "failed")
		assert.Len(t, values, 3)
	})

	t.Run("PlanStatus values", func(t *testing.T) {
		values := PlanStatusPending.Values()
		assert.Contains(t, values, "pending")
		assert.Contains(t, values, "executing")
		assert.Contains(t, values, "completed")
		assert.Contains(t, values, "failed")
		assert.Len(t, values, 4)
	})
}
