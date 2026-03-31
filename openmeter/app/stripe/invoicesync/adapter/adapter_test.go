package invoicesyncadapter

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type testSetup struct {
	adapter  *Adapter
	dbClient *entdb.Client
	dbURL    string
}

func setupTestAdapter(t testing.TB) *testSetup {
	t.Helper()

	testDB := testutils.InitPostgresDB(t)
	t.Cleanup(func() {
		_ = testDB.EntDriver.Close()
		_ = testDB.PGDriver.Close()
	})

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		srcErr, dbErr := migrator.Close()
		_ = srcErr
		_ = dbErr
	})
	require.NoError(t, migrator.Up())

	dbClient := testDB.EntDriver.Client()

	adapter, err := New(Config{Client: dbClient})
	require.NoError(t, err)

	// Drop FK constraint for testing without full billing stack
	dropFKConstraint(t, testDB.URL)

	return &testSetup{
		adapter:  adapter,
		dbClient: dbClient,
		dbURL:    testDB.URL,
	}
}

// dropFKConstraint drops the FK constraint from app_stripe_invoice_sync_plans so we can test
// without setting up the full billing invoice dependency chain.
func dropFKConstraint(t testing.TB, dbURL string) {
	t.Helper()

	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err)
	defer db.Close()

	var constraintName string
	err = db.QueryRow(`
		SELECT c.conname
		FROM pg_constraint c
		JOIN pg_class rel ON rel.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = rel.relnamespace
		JOIN pg_class frel ON frel.oid = c.confrelid
		WHERE n.nspname = current_schema()
		  AND rel.relname = 'app_stripe_invoice_sync_plans'
		  AND c.contype = 'f'
		  AND frel.relname = 'billing_invoices'
		LIMIT 1
	`).Scan(&constraintName)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	require.NoError(t, err)

	_, err = db.Exec(fmt.Sprintf(
		`ALTER TABLE app_stripe_invoice_sync_plans DROP CONSTRAINT %s`,
		quoteIdentifier(constraintName),
	))
	require.NoError(t, err)
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// testInvoiceID returns a random invoice ID for testing.
func testInvoiceID() string {
	return ulid.Make().String()
}

func TestCreateAndGetSyncPlan(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()
	namespace := "ns-test"

	invoiceID := testInvoiceID()

	payload1, _ := json.Marshal(invoicesync.InvoiceCreatePayload{
		AppID:            "app-1",
		Namespace:        namespace,
		CustomerID:       "cust-1",
		InvoiceID:        invoiceID,
		StripeCustomerID: "cus_stripe",
		Currency:         "USD",
	})
	payload2, _ := json.Marshal(invoicesync.LineItemAddPayload{
		StripeInvoiceID: "",
		Lines: []invoicesync.LineItemParams{
			{Description: "Test", Amount: 1000, Currency: "USD"},
		},
	})

	plan := invoicesync.SyncPlan{
		Namespace: namespace,
		InvoiceID: invoiceID,
		AppID:     "app-test-1",
		SessionID: fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		Phase:     invoicesync.SyncPlanPhaseDraft,
		Operations: []invoicesync.SyncOperation{
			{
				Sequence:       0,
				Type:           invoicesync.OpTypeInvoiceCreate,
				Payload:        payload1,
				IdempotencyKey: "key-0",
				Status:         invoicesync.OpStatusPending,
			},
			{
				Sequence:       1,
				Type:           invoicesync.OpTypeLineItemAdd,
				Payload:        payload2,
				IdempotencyKey: "key-1",
				Status:         invoicesync.OpStatusPending,
			},
		},
	}

	// Create
	created, err := setup.adapter.CreateSyncPlan(ctx, plan)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, invoicesync.PlanStatusPending, created.Status)
	assert.Len(t, created.Operations, 2)
	assert.NotEmpty(t, created.Operations[0].ID)
	assert.NotEmpty(t, created.Operations[1].ID)
	assert.Equal(t, created.ID, created.Operations[0].PlanID)

	// Get by ID
	fetched, err := setup.adapter.GetSyncPlan(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, plan.Namespace, fetched.Namespace)
	assert.Equal(t, plan.InvoiceID, fetched.InvoiceID)
	assert.Equal(t, plan.SessionID, fetched.SessionID)
	assert.Equal(t, invoicesync.SyncPlanPhaseDraft, fetched.Phase)
	assert.Len(t, fetched.Operations, 2)

	// Operations should be ordered by sequence
	assert.Equal(t, 0, fetched.Operations[0].Sequence)
	assert.Equal(t, 1, fetched.Operations[1].Sequence)
	assert.Equal(t, invoicesync.OpTypeInvoiceCreate, fetched.Operations[0].Type)
	assert.Equal(t, invoicesync.OpTypeLineItemAdd, fetched.Operations[1].Type)
}

// helper to create a sync plan with a real invoice parent
func createPlan(t testing.TB, setup *testSetup, namespace string, phase invoicesync.SyncPlanPhase, ops []invoicesync.SyncOperation) invoicesync.SyncPlan {
	t.Helper()
	invoiceID := testInvoiceID()
	plan, err := setup.adapter.CreateSyncPlan(context.Background(), invoicesync.SyncPlan{
		Namespace:  namespace,
		InvoiceID:  invoiceID,
		AppID:      "app-test-" + ulid.Make().String(),
		SessionID:  fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		Phase:      phase,
		Operations: ops,
	})
	require.NoError(t, err)
	return plan
}

func testPayload() json.RawMessage {
	p, _ := json.Marshal(invoicesync.InvoiceUpdatePayload{StripeInvoiceID: "in_123"})
	return p
}

func TestGetActiveSyncPlanByInvoice(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()
	ns := "ns-test"
	invoiceID := testInvoiceID()

	plan, err := setup.adapter.CreateSyncPlan(ctx, invoicesync.SyncPlan{
		Namespace: ns, InvoiceID: invoiceID, AppID: "app-test-1",
		SessionID: fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		Phase:     invoicesync.SyncPlanPhaseDraft,
		Operations: []invoicesync.SyncOperation{
			{Sequence: 0, Type: invoicesync.OpTypeInvoiceCreate, Payload: testPayload(), IdempotencyKey: "k1"},
		},
	})
	require.NoError(t, err)

	found, err := setup.adapter.GetActiveSyncPlanByInvoice(ctx, ns, invoiceID, invoicesync.SyncPlanPhaseDraft)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, plan.ID, found.ID)

	notFound, err := setup.adapter.GetActiveSyncPlanByInvoice(ctx, ns, invoiceID, invoicesync.SyncPlanPhaseIssuing)
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestGetNextPendingOperation(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceCreate, Payload: testPayload(), IdempotencyKey: "k0"},
		{Sequence: 1, Type: invoicesync.OpTypeLineItemAdd, Payload: testPayload(), IdempotencyKey: "k1"},
	})

	op, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, 0, op.Sequence)

	resp, _ := json.Marshal(map[string]string{"id": "in_new"})
	require.NoError(t, setup.adapter.CompleteOperation(ctx, op.ID, resp))

	op, err = setup.adapter.GetNextPendingOperation(ctx, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, 1, op.Sequence)

	require.NoError(t, setup.adapter.CompleteOperation(ctx, op.ID, resp))

	op, err = setup.adapter.GetNextPendingOperation(ctx, plan.ID)
	require.NoError(t, err)
	assert.Nil(t, op)
}

func TestCompleteOperation(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceCreate, Payload: testPayload(), IdempotencyKey: "k0"},
	})

	respData, _ := json.Marshal(invoicesync.InvoiceCreateResponse{
		StripeInvoiceID: "in_created", InvoiceNumber: "INV-001",
	})
	require.NoError(t, setup.adapter.CompleteOperation(ctx, plan.Operations[0].ID, respData))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.OpStatusCompleted, fetched.Operations[0].Status)
	assert.NotNil(t, fetched.Operations[0].CompletedAt)

	var stored invoicesync.InvoiceCreateResponse
	require.NoError(t, json.Unmarshal(fetched.Operations[0].StripeResponse, &stored))
	assert.Equal(t, "in_created", stored.StripeInvoiceID)
}

func TestFailOperation(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: testPayload(), IdempotencyKey: "k0"},
	})

	require.NoError(t, setup.adapter.FailOperation(ctx, plan.Operations[0].ID, "stripe returned 400"))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.OpStatusFailed, fetched.Operations[0].Status)
	require.NotNil(t, fetched.Operations[0].Error)
	assert.Equal(t, "stripe returned 400", *fetched.Operations[0].Error)
}

func TestCompletePlan(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDelete, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceDelete, Payload: testPayload(), IdempotencyKey: "k0"},
	})

	require.NoError(t, setup.adapter.CompletePlan(ctx, plan.ID))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.PlanStatusCompleted, fetched.Status)
	assert.NotNil(t, fetched.CompletedAt)
}

func TestFailPlan(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: testPayload(), IdempotencyKey: "k0"},
		{Sequence: 1, Type: invoicesync.OpTypeLineItemAdd, Payload: testPayload(), IdempotencyKey: "k1"},
	})

	require.NoError(t, setup.adapter.FailPlan(ctx, plan.ID, "fatal error"))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.PlanStatusFailed, fetched.Status)
	require.NotNil(t, fetched.Error)
	assert.Equal(t, "fatal error", *fetched.Error)
	assert.NotNil(t, fetched.CompletedAt)

	// Both pending operations should have been canceled
	for _, op := range fetched.Operations {
		assert.Equal(t, invoicesync.OpStatusFailed, op.Status,
			"pending op %d should be failed", op.Sequence)
		require.NotNil(t, op.Error)
		assert.Contains(t, *op.Error, "plan failed")
		assert.NotNil(t, op.CompletedAt)
	}

	// No pending operations should remain
	nextOp, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
	require.NoError(t, err)
	assert.Nil(t, nextOp, "no pending operations should remain after FailPlan")
}

func TestGetSyncPlanNotFound(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan, err := setup.adapter.GetSyncPlan(ctx, "nonexistent-id-00000000000")
	require.NoError(t, err)
	assert.Nil(t, plan)
}

func TestUpdatePlanStatus(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-test", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceCreate, Payload: testPayload(), IdempotencyKey: "k0"},
	})
	assert.Equal(t, invoicesync.PlanStatusPending, plan.Status)

	require.NoError(t, setup.adapter.UpdatePlanStatus(ctx, plan.ID, invoicesync.PlanStatusExecuting, nil))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.PlanStatusExecuting, fetched.Status)
	assert.Nil(t, fetched.CompletedAt)
}

// TestGetNextPendingOperationConcurrency verifies that concurrent callers can safely drain a plan's
// operations without panics or data races. The adapter does not use row-level locking, so multiple
// goroutines may claim the same pending operation — that's fine because the handler uses an advisory
// lock to serialize execution. This test only checks safety (no panics/races) and eventual completion.
func TestGetNextPendingOperationConcurrency(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	const numOps = 10
	ops := make([]invoicesync.SyncOperation, numOps)
	for i := 0; i < numOps; i++ {
		ops[i] = invoicesync.SyncOperation{
			Sequence:       i,
			Type:           invoicesync.OpTypeInvoiceUpdate,
			Payload:        testPayload(),
			IdempotencyKey: fmt.Sprintf("conc-k%d", i),
			Status:         invoicesync.OpStatusPending,
		}
	}
	plan := createPlan(t, setup, "ns-concurrency", invoicesync.SyncPlanPhaseDraft, ops)

	resp, _ := json.Marshal(map[string]string{"id": "in_test"})

	var wg sync.WaitGroup
	for g := 0; g < numOps; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
			if err != nil || op == nil {
				return
			}
			_ = setup.adapter.CompleteOperation(ctx, op.ID, resp)
		}()
	}
	wg.Wait()

	// Drain any operations that were not claimed by the concurrent goroutines.
	for {
		op, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
		require.NoError(t, err)
		if op == nil {
			break
		}
		require.NoError(t, setup.adapter.CompleteOperation(ctx, op.ID, resp))
	}

	// Every operation must now be completed and each sequence unique.
	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Operations, numOps)
	seenSeqs := map[int]bool{}
	for _, op := range fetched.Operations {
		assert.Equal(t, invoicesync.OpStatusCompleted, op.Status,
			"op sequence %d should be completed", op.Sequence)
		assert.False(t, seenSeqs[op.Sequence],
			"sequence %d appears more than once in the plan", op.Sequence)
		seenSeqs[op.Sequence] = true
	}
}

// BenchmarkGetNextAndCompleteOperation creates a plan with 100 operations and measures the
// sequential GetNextPendingOperation + CompleteOperation throughput.
func BenchmarkGetNextAndCompleteOperation(b *testing.B) {
	setup := setupTestAdapter(b)
	ctx := context.Background()
	resp, _ := json.Marshal(map[string]string{"id": "in_bench"})

	b.ResetTimer()
	for iter := range b.N {
		b.StopTimer()
		const numOps = 100
		ops := make([]invoicesync.SyncOperation, numOps)
		for i := 0; i < numOps; i++ {
			ops[i] = invoicesync.SyncOperation{
				Sequence:       i,
				Type:           invoicesync.OpTypeInvoiceUpdate,
				Payload:        testPayload(),
				IdempotencyKey: fmt.Sprintf("bench-k%d-%d", iter, i),
				Status:         invoicesync.OpStatusPending,
			}
		}
		plan := createPlan(b, setup, "ns-bench", invoicesync.SyncPlanPhaseDraft, ops)
		b.StartTimer()

		for {
			op, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
			if err != nil || op == nil {
				break
			}
			if err := setup.adapter.CompleteOperation(ctx, op.ID, resp); err != nil {
				b.Fatalf("CompleteOperation: %v", err)
			}
		}
	}
}

// TestCreateSyncPlan_LargePayload verifies that operations with large JSON payloads round-trip
// correctly through the database without truncation.
func TestCreateSyncPlan_LargePayload(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	// Build a payload ~64 KiB.
	large := bytes.Repeat([]byte("x"), 64*1024)
	rawPayload, err := json.Marshal(map[string]string{
		"blob": string(large),
	})
	require.NoError(t, err)

	plan := createPlan(t, setup, "ns-large", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: rawPayload, IdempotencyKey: "large-k0"},
	})

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Operations, 1)
	assert.JSONEq(t, string(rawPayload), string(fetched.Operations[0].Payload))
	// Verify no truncation: fetched payload should be at least as large as the input.
	assert.GreaterOrEqual(t, len(fetched.Operations[0].Payload), len(rawPayload))
}

// TestCreateSyncPlan_DuplicateIdempotencyKey verifies that creating two plans that happen to share
// an idempotency key value on their operations is permitted (the key is advisory, not a DB unique
// constraint across plans).
func TestCreateSyncPlan_DuplicateIdempotencyKey(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	sharedKey := "shared-idem-key"

	plan1 := createPlan(t, setup, "ns-idem", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: testPayload(), IdempotencyKey: sharedKey},
	})
	plan2 := createPlan(t, setup, "ns-idem", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: testPayload(), IdempotencyKey: sharedKey},
	})

	// Both plans must be retrievable independently.
	for _, id := range []string{plan1.ID, plan2.ID} {
		fetched, err := setup.adapter.GetSyncPlan(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Len(t, fetched.Operations, 1)
		assert.Equal(t, sharedKey, fetched.Operations[0].IdempotencyKey)
	}
}

// TestRapidUpdatePlanStatusAndFailPlan verifies that rapid status transitions do not leave the plan
// or its operations in an inconsistent state and do not panic.
func TestRapidUpdatePlanStatusAndFailPlan(t *testing.T) {
	setup := setupTestAdapter(t)
	ctx := context.Background()

	plan := createPlan(t, setup, "ns-rapid", invoicesync.SyncPlanPhaseDraft, []invoicesync.SyncOperation{
		{Sequence: 0, Type: invoicesync.OpTypeInvoiceUpdate, Payload: testPayload(), IdempotencyKey: "rapid-k0"},
		{Sequence: 1, Type: invoicesync.OpTypeLineItemAdd, Payload: testPayload(), IdempotencyKey: "rapid-k1"},
	})

	// Transition through intermediate statuses before failing.
	require.NoError(t, setup.adapter.UpdatePlanStatus(ctx, plan.ID, invoicesync.PlanStatusExecuting, nil))
	require.NoError(t, setup.adapter.UpdatePlanStatus(ctx, plan.ID, invoicesync.PlanStatusPending, nil))
	require.NoError(t, setup.adapter.FailPlan(ctx, plan.ID, "rapid failure"))

	fetched, err := setup.adapter.GetSyncPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, invoicesync.PlanStatusFailed, fetched.Status)
	require.NotNil(t, fetched.Error)
	assert.Equal(t, "rapid failure", *fetched.Error)

	// All pending operations must have been canceled.
	for _, op := range fetched.Operations {
		assert.Equal(t, invoicesync.OpStatusFailed, op.Status,
			"op sequence %d should be failed after FailPlan", op.Sequence)
	}

	// No more pending operations remain.
	nextOp, err := setup.adapter.GetNextPendingOperation(ctx, plan.ID)
	require.NoError(t, err)
	assert.Nil(t, nextOp)
}
