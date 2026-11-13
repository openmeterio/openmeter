package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type invoicePayloadRecorder struct {
	notification.Service
	payload notification.EventPayload
}

func (*invoicePayloadRecorder) ListRules(context.Context, notification.ListRulesInput) (notification.ListRulesResult, error) {
	return notification.ListRulesResult{Items: []notification.Rule{{}}}, nil
}

func (r *invoicePayloadRecorder) CreateEvent(_ context.Context, input notification.CreateEventInput) (*notification.Event, error) {
	r.payload = input.Payload
	return &notification.Event{}, nil
}

func TestInvoiceNotificationPayload(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{From: created, To: created.AddDate(0, 1, 0)}
	invoice := billing.StandardInvoiceBase{
		ID:        "invoice-1",
		Namespace: "ns",
		Number:    "INV-1",
		Currency:  "USD",
		CreatedAt: created,
		UpdatedAt: created,
		Type:      billing.InvoiceTypeStandard,
		Status:    billing.StandardInvoiceStatusDraftCreated,
		Customer:  billing.InvoiceCustomer{CustomerID: "customer-1", Name: "Test Customer"},
		Workflow: billing.InvoiceWorkflow{
			Config:        billing.WorkflowConfig{Collection: billing.CollectionConfig{Alignment: billing.AlignmentKindSubscription}},
			AppReferences: billing.ProfileAppReferences{Payment: app.AppID{ID: "profile-payment"}},
		},
	}
	createdEvent := billing.EventStandardInvoice{Invoice: billing.StandardInvoice{StandardInvoiceBase: invoice}}
	updated := createdEvent
	var lines []*billing.StandardLine
	for _, name := range []string{"Storage", "Compute"} {
		line := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
			ID:            name,
			Name:          name,
			InvoiceID:     invoice.ID,
			Currency:      invoice.Currency,
			CreatedAt:     created,
			UpdatedAt:     created,
			InvoiceAt:     period.To,
			Period:        period,
			PerUnitAmount: alpacadecimal.NewFromFloat(12.50),
			PaymentTerm:   productcatalog.InAdvancePaymentTerm,
		})
		lines = append(lines, line)
	}
	lines[0].CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(2), Description: "credit grant"}}
	updated.Invoice.Lines = billing.NewStandardInvoiceLines(lines)
	updated.Invoice.CollectionAt = &period.To
	updated.Apps.Payment.Type = app.AppTypeSandbox
	updated.Apps.Payment.ID = "snapshot-payment"
	updated.Apps.Payment.Name = "Payment app at event time"

	for _, tc := range []struct {
		eventType notification.EventType
		event     billing.EventStandardInvoice
	}{
		{notification.EventTypeInvoiceCreated, createdEvent},
		{notification.EventTypeInvoiceUpdated, updated},
	} {
		t.Run(string(tc.eventType), func(t *testing.T) {
			recorder := &invoicePayloadRecorder{}
			handler := InvoiceEventHandler{Notification: recorder, Logger: slog.New(slog.DiscardHandler)}
			require.NoError(t, handler.Handle(t.Context(), tc.event, tc.eventType))

			got, err := json.Marshal(recorder.payload)
			require.NoError(t, err)
			path := filepath.Join("testdata", string(tc.eventType)+".json")
			// These payloads were recorded through the consumer before extracting the HTTP mappers.
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, string(want), string(got))
		})
	}
}
