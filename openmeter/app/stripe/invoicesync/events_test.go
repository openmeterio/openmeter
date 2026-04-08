package invoicesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteSyncPlanEvent_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{
			PlanID: "plan-1", InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
		}.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing plan_id", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{
			InvoiceID: "inv-1", Namespace: "ns", CustomerID: "c-1",
		}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plan_id")
	})

	t.Run("missing invoice_id", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{
			PlanID: "plan-1", Namespace: "ns", CustomerID: "c-1",
		}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invoice_id")
	})

	t.Run("missing namespace", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{
			PlanID: "plan-1", InvoiceID: "inv-1", CustomerID: "c-1",
		}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespace")
	})

	t.Run("missing customer_id", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{
			PlanID: "plan-1", InvoiceID: "inv-1", Namespace: "ns",
		}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "customer_id")
	})

	t.Run("all missing", func(t *testing.T) {
		err := ExecuteSyncPlanEvent{}.Validate()
		require.Error(t, err)
	})
}

func TestExecuteSyncPlanEvent_EventName(t *testing.T) {
	e := ExecuteSyncPlanEvent{PlanID: "p", InvoiceID: "i", Namespace: "n", CustomerID: "c"}
	name := e.EventName()
	assert.NotEmpty(t, name)
	assert.Contains(t, name, "sync_plan")
}

func TestExecuteSyncPlanEvent_EventMetadata(t *testing.T) {
	e := ExecuteSyncPlanEvent{PlanID: "plan-1", InvoiceID: "inv-1", Namespace: "ns-1", CustomerID: "c-1"}
	meta := e.EventMetadata()
	assert.Contains(t, meta.Source, "ns-1")
	assert.Contains(t, meta.Source, "inv-1")
	assert.Contains(t, meta.Subject, "plan-1")
}
