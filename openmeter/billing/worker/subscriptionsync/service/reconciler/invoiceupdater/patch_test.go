package invoiceupdater

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNewUpdateLinePatchInputValidate(t *testing.T) {
	t.Parallel()

	period := updateLinePatchTestPeriod()
	validIdentity := NewUpdateLinePatchInput{
		Line: billing.LineID{
			Namespace: "ns",
			ID:        "line-1",
		},
		InvoiceID: "invoice-1",
	}

	testCases := []struct {
		name        string
		input       NewUpdateLinePatchInput
		errorString string
	}{
		{
			name: "service period update",
			input: func() NewUpdateLinePatchInput {
				input := validIdentity
				input.ServicePeriod = mo.Some(period)
				return input
			}(),
		},
		{
			name: "explicit deleted at clear",
			input: func() NewUpdateLinePatchInput {
				input := validIdentity
				input.DeletedAt = mo.Some[*time.Time](nil)
				return input
			}(),
		},
		{
			name:        "empty update",
			input:       validIdentity,
			errorString: "at least one line update is required",
		},
		{
			name: "missing identity",
			input: NewUpdateLinePatchInput{
				ServicePeriod: mo.Some(period),
			},
			errorString: "invoice id is required",
		},
		{
			name: "zero invoice at",
			input: func() NewUpdateLinePatchInput {
				input := validIdentity
				input.InvoiceAt = mo.Some(time.Time{})
				return input
			}(),
			errorString: "invoice at is required",
		},
		{
			name: "negative flat fee per unit amount",
			input: func() NewUpdateLinePatchInput {
				input := validIdentity
				input.FlatFeePerUnitAmount = mo.Some(alpacadecimal.NewFromInt(-1))
				return input
			}(),
			errorString: "flat fee per unit amount must not be negative",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()
			if tt.errorString == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.errorString)
		})
	}
}

func TestNewUpdateLinePatchInputIsNoop(t *testing.T) {
	t.Parallel()

	require.True(t, NewUpdateLinePatchInput{}.IsNoop())
	require.True(t, NewUpdateLinePatchInput{
		Line: billing.LineID{
			Namespace: "ns",
			ID:        "line-1",
		},
		InvoiceID: "invoice-1",
	}.IsNoop())
	require.False(t, NewUpdateLinePatchInput{ServicePeriod: mo.Some(updateLinePatchTestPeriod())}.IsNoop())
	require.False(t, NewUpdateLinePatchInput{InvoiceAt: mo.Some(time.Time{})}.IsNoop())
	require.False(t, NewUpdateLinePatchInput{DeletedAt: mo.Some[*time.Time](nil)}.IsNoop())
	require.False(t, NewUpdateLinePatchInput{FlatFeePerUnitAmount: mo.Some(alpacadecimal.Zero)}.IsNoop())
}

func TestPatchLineUpdateApplyGatheringLine(t *testing.T) {
	t.Parallel()

	t.Run("omitted fields are preserved", func(t *testing.T) {
		t.Parallel()

		line := newUpdateLinePatchTestGatheringLine()
		originalInvoiceAt := line.InvoiceAt
		originalDeletedAt := *line.DeletedAt
		originalDBState := line.DBState
		updatedPeriod := line.ServicePeriod
		updatedPeriod.To = updatedPeriod.To.AddDate(0, 1, 0)

		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:          line.GetLineID(),
			InvoiceID:     line.InvoiceID,
			ServicePeriod: mo.Some(updatedPeriod),
		})
		require.NoError(t, err)

		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)
		updatedGeneric, err := update.Apply(line.AsGenericLine())
		require.NoError(t, err)
		updated, err := updatedGeneric.AsInvoiceLine().AsGatheringLine()
		require.NoError(t, err)

		require.Equal(t, updatedPeriod, updated.ServicePeriod)
		require.Equal(t, originalInvoiceAt, updated.InvoiceAt)
		require.Equal(t, originalDeletedAt, *updated.DeletedAt)
		require.Same(t, originalDBState, updated.DBState)
		require.Equal(t, "preserved", updated.Metadata["owner"])
		require.NotNil(t, updated.SplitLineHierarchy)
		require.NotSame(t, line.SplitLineHierarchy, updated.SplitLineHierarchy)
		require.Equal(t, "group-1", updated.SplitLineHierarchy.Group.ID)

		amount, err := GetFlatFeePerUnitAmount(updated.AsGenericLine())
		require.NoError(t, err)
		require.True(t, alpacadecimal.NewFromInt(10).Equal(amount))

		updated.Metadata["owner"] = "changed"
		require.Equal(t, "preserved", line.Metadata["owner"])
		require.NotEqual(t, updatedPeriod, line.ServicePeriod)
	})

	t.Run("present nil clears deleted at", func(t *testing.T) {
		t.Parallel()

		line := newUpdateLinePatchTestGatheringLine()
		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:      line.GetLineID(),
			InvoiceID: line.InvoiceID,
			DeletedAt: mo.Some[*time.Time](nil),
		})
		require.NoError(t, err)

		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)
		updatedGeneric, err := update.Apply(line.AsGenericLine())
		require.NoError(t, err)
		updated, err := updatedGeneric.AsInvoiceLine().AsGatheringLine()
		require.NoError(t, err)

		require.Nil(t, updated.DeletedAt)
		require.NotNil(t, line.DeletedAt)
		require.Equal(t, line.ServicePeriod, updated.ServicePeriod)
		require.Equal(t, line.InvoiceAt, updated.InvoiceAt)
	})

	t.Run("flat fee amount preserves payment term", func(t *testing.T) {
		t.Parallel()

		line := newUpdateLinePatchTestGatheringLine()
		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:                 line.GetLineID(),
			InvoiceID:            line.InvoiceID,
			FlatFeePerUnitAmount: mo.Some(alpacadecimal.NewFromInt(25)),
		})
		require.NoError(t, err)

		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)
		updatedGeneric, err := update.Apply(line.AsGenericLine())
		require.NoError(t, err)
		updated, err := updatedGeneric.AsInvoiceLine().AsGatheringLine()
		require.NoError(t, err)
		flatPrice, err := updated.Price.AsFlat()
		require.NoError(t, err)

		require.True(t, alpacadecimal.NewFromInt(25).Equal(flatPrice.Amount))
		require.Equal(t, productcatalog.InAdvancePaymentTerm, flatPrice.PaymentTerm)
	})
}

func TestPatchLineUpdateApplyRejectsIncompatibleFields(t *testing.T) {
	t.Parallel()

	line := newUpdateLinePatchTestStandardLine()

	t.Run("invoice at on standard line", func(t *testing.T) {
		t.Parallel()

		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:      line.GetLineID(),
			InvoiceID: line.InvoiceID,
			InvoiceAt: mo.Some(line.InvoiceAt.Add(time.Hour)),
		})
		require.NoError(t, err)
		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)

		_, err = update.Apply(line.AsGenericLine())
		require.ErrorContains(t, err, "does not support invoice at updates")
	})

	t.Run("flat fee amount on usage price", func(t *testing.T) {
		t.Parallel()

		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:                 line.GetLineID(),
			InvoiceID:            line.InvoiceID,
			FlatFeePerUnitAmount: mo.Some(alpacadecimal.NewFromInt(2)),
		})
		require.NoError(t, err)
		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)

		_, err = update.Apply(line.AsGenericLine())
		require.ErrorContains(t, err, "setting flat fee per unit amount")
	})
}

func TestPatchLineUpdateApplyStandardLinePreservesOmittedFields(t *testing.T) {
	t.Parallel()

	line := newUpdateLinePatchTestStandardLine()
	line.Metadata = models.Metadata{"owner": "preserved"}
	line.DBState = &billing.StandardLine{}
	originalDBState := line.DBState
	originalInvoiceAt := line.InvoiceAt
	updatedPeriod := line.Period
	updatedPeriod.To = updatedPeriod.From.AddDate(0, 0, 15)
	patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
		Line:          line.GetLineID(),
		InvoiceID:     line.InvoiceID,
		ServicePeriod: mo.Some(updatedPeriod),
	})
	require.NoError(t, err)
	update, err := patch.AsUpdateLinePatch()
	require.NoError(t, err)

	updatedGeneric, err := update.Apply(line.AsGenericLine())
	require.NoError(t, err)
	updated, err := updatedGeneric.AsInvoiceLine().AsStandardLine()
	require.NoError(t, err)

	require.Equal(t, updatedPeriod, updated.Period)
	require.Equal(t, originalInvoiceAt, updated.InvoiceAt)
	require.Equal(t, models.Metadata{"owner": "preserved"}, updated.Metadata)
	require.Same(t, originalDBState, updated.DBState)
	require.NotEqual(t, updatedPeriod, line.Period)
}

func TestParsePatchesGroupsUpdateByInvoiceID(t *testing.T) {
	t.Parallel()

	period := updateLinePatchTestPeriod()
	first, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
		Line: billing.LineID{
			Namespace: "ns",
			ID:        "line-1",
		},
		InvoiceID:     "invoice-1",
		ServicePeriod: mo.Some(period),
	})
	require.NoError(t, err)
	second, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
		Line: billing.LineID{
			Namespace: "ns",
			ID:        "line-2",
		},
		InvoiceID: "invoice-2",
		DeletedAt: mo.Some[*time.Time](nil),
	})
	require.NoError(t, err)

	parsed, err := (&Updater{}).parsePatches([]Patch{first, second})
	require.NoError(t, err)
	require.Len(t, parsed.updatedLinesByInvoiceID, 2)
	require.Equal(t, "line-1", parsed.updatedLinesByInvoiceID["invoice-1"].updatedLines[0].Line.ID)
	require.Equal(t, "line-2", parsed.updatedLinesByInvoiceID["invoice-2"].updatedLines[0].Line.ID)
}

func TestUpdateLinePatchLogDistinguishesOmittedFromClear(t *testing.T) {
	t.Parallel()

	patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
		Line: billing.LineID{
			Namespace: "ns",
			ID:        "line-1",
		},
		InvoiceID: "invoice-1",
		DeletedAt: mo.Some[*time.Time](nil),
	})
	require.NoError(t, err)

	var output bytes.Buffer
	patch.Log(slog.New(slog.NewJSONHandler(&output, nil)))

	entry := map[string]any{}
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	require.Equal(t, []any{"deleted_at"}, entry["updated_fields"])
	require.Contains(t, entry, "new_deleted_at")
	require.Nil(t, entry["new_deleted_at"])
	require.NotContains(t, entry, "new_service_period_from")
	require.NotContains(t, entry, "unique_reference_id")
}

func updateLinePatchTestPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newUpdateLinePatchTestGatheringLine() billing.GatheringLine {
	period := updateLinePatchTestPeriod()
	deletedAt := period.From.Add(time.Hour)

	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-1",
				Name:      "line",
			}),
			Metadata:      models.Metadata{"owner": "preserved"},
			ManagedBy:     billing.SubscriptionManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			InvoiceID:     "invoice-1",
			Currency:      currencyx.FiatCode("USD"),
			ServicePeriod: period,
			InvoiceAt:     period.To,
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		DBState: &billing.GatheringLine{},
		SplitLineHierarchy: &billing.SplitLineHierarchy{
			Group: billing.SplitLineGroup{
				NamespacedID: models.NamespacedID{
					Namespace: "ns",
					ID:        "group-1",
				},
			},
		},
	}
	line.DeletedAt = &deletedAt

	return line
}

func newUpdateLinePatchTestStandardLine() *billing.StandardLine {
	period := updateLinePatchTestPeriod()

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "line-1",
				Name:      "line",
			}),
			ManagedBy: billing.SubscriptionManagedLine,
			Engine:    billing.LineEngineTypeInvoice,
			InvoiceID: "invoice-1",
			Currency:  currencyx.FiatCode("USD"),
			Period:    period,
			InvoiceAt: period.To,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			FeatureKey: "feature-key",
		},
	}
}
