package invoiceupdater

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPatchesGetSingularStandardLineUpdate(t *testing.T) {
	target := newValidStandardLinePatchTarget()

	line, err := Patches{
		NewUpdateLinePatch(target.AsGenericLine()),
	}.GetSingularStandardLineUpdate(target.GetLineID(), target.InvoiceID)

	require.NoError(t, err)
	require.Equal(t, target, line)
}

func TestPatchesGetSingularStandardLineUpdateAllowsNoUpdate(t *testing.T) {
	target := newValidStandardLinePatchTarget()

	line, err := Patches(nil).GetSingularStandardLineUpdate(target.GetLineID(), target.InvoiceID)

	require.NoError(t, err)
	require.Nil(t, line)
}

func TestPatchesGetSingularStandardLineUpdateRejectsInvalidPatches(t *testing.T) {
	target := newValidStandardLinePatchTarget()

	wrongLine, err := target.Clone()
	require.NoError(t, err)
	wrongLine.ID = "other-line"

	wrongInvoice, err := target.Clone()
	require.NoError(t, err)
	wrongInvoice.InvoiceID = "other-invoice"

	invalidTarget, err := target.Clone()
	require.NoError(t, err)
	invalidTarget.UsageBased = nil

	gatheringTarget := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: target.ManagedResource,
			InvoiceID:       target.InvoiceID,
		},
	}

	tests := []struct {
		name        string
		patches     Patches
		errContains string
	}{
		{
			name: "multiple patches",
			patches: Patches{
				NewUpdateLinePatch(target.AsGenericLine()),
				NewUpdateLinePatch(target.AsGenericLine()),
			},
			errContains: "expected singular standard line update patch",
		},
		{
			name: "wrong operation",
			patches: Patches{
				NewDeleteLinePatch(target.GetLineID(), target.InvoiceID),
			},
			errContains: "expected update line patch",
		},
		{
			name: "missing target state",
			patches: Patches{
				NewUpdateLinePatch(nil),
			},
			errContains: "target state is required",
		},
		{
			name: "wrong line",
			patches: Patches{
				NewUpdateLinePatch(wrongLine.AsGenericLine()),
			},
			errContains: "does not match line",
		},
		{
			name: "wrong invoice",
			patches: Patches{
				NewUpdateLinePatch(wrongInvoice.AsGenericLine()),
			},
			errContains: "does not match invoice",
		},
		{
			name: "gathering line target",
			patches: Patches{
				NewUpdateLinePatch(gatheringTarget.AsGenericLine()),
			},
			errContains: "target state must be a standard line",
		},
		{
			name: "invalid standard line target",
			patches: Patches{
				NewUpdateLinePatch(invalidTarget.AsGenericLine()),
			},
			errContains: "validating target standard line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.patches.GetSingularStandardLineUpdate(target.GetLineID(), target.InvoiceID)

			require.ErrorContains(t, err, tt.errContains)
		})
	}
}

func newValidStandardLinePatchTarget() *billing.StandardLine {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "test-namespace",
				},
				ID:   "line-1",
				Name: "flat fee",
			},
			ManagedBy: billing.SystemManagedLine,
			Engine:    billing.LineEngineTypeChargeFlatFee,
			InvoiceID: "invoice-1",
			Currency:  currencyx.FiatCode("USD"),
			Period: timeutil.ClosedPeriod{
				From: start,
				To:   start.Add(time.Hour),
			},
			InvoiceAt: start.Add(time.Hour),
			Totals: totals.Totals{
				Total: alpacadecimal.NewFromInt(1),
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
}
