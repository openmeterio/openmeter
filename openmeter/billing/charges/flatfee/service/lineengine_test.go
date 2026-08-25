package service

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestValidateCustomCurrencyInvoiceLineDeleteAllowsDraftInvoice(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newFlatFeeStandardLineForTest(servicePeriod)
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)
	run := newFlatFeeCustomCurrencyRunForTest(servicePeriod, line.Totals, false)
	run.LineID = lo.ToPtr(line.ID)
	run.InvoiceID = lo.ToPtr(line.InvoiceID)
	charge.Realizations.CurrentRun = &run

	invoice := &billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: line.Namespace,
			ID:        line.InvoiceID,
			Status:    billing.StandardInvoiceStatusDraftCreated,
		},
	}

	require.NoError(t, validateCustomCurrencyInvoiceLineDelete(invoice, line.AsGenericLine(), charge))
}
