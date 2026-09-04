package billingservice

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestAreGatheringLinesBillableAsOfPreservesInputOrderAcrossEngines(t *testing.T) {
	invoiceEngine := &billabilityLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
	}
	chargeEngine := &billabilityLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeChargeUsageBased},
	}

	svc := &Service{lineEngines: newEngineRegistry()}
	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	lines := billing.GatheringLines{
		newGatheringLineForLineEngineTest("invoice-1", billing.LineEngineTypeInvoice, false),
		newGatheringLineForLineEngineTest("charge-1", billing.LineEngineTypeChargeUsageBased, false),
		newGatheringLineForLineEngineTest("invoice-2", billing.LineEngineTypeInvoice, false),
	}
	invoice := billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "ns"},
			ID:              "invoice-1",
		},
	}}

	results, err := svc.areGatheringLinesBillableAsOf(t.Context(), billing.AreLinesBillableAsOfInput{
		Invoice: invoice,
		AsOf:    lines[0].InvoiceAt,
		Lines:   lines,
	})
	require.NoError(t, err)
	require.Same(t, invoiceEngine, results[0].Engine)
	require.Same(t, chargeEngine, results[1].Engine)
	require.Same(t, invoiceEngine, results[2].Engine)
	require.Equal(t, []string{"invoice-1", "invoice-2"}, []string{
		invoiceEngine.inputs[0].Lines[0].ID,
		invoiceEngine.inputs[0].Lines[1].ID,
	})
	require.Equal(t, "charge-1", chargeEngine.inputs[0].Lines[0].ID)
}

func TestCheckIfGatheringLinesAreInvoicableUsesEachLineInvoiceAt(t *testing.T) {
	engine := &billabilityLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
	}
	svc := &Service{lineEngines: newEngineRegistry()}
	require.NoError(t, svc.RegisterLineEngine(engine))

	// Given gathering lines with different scheduled invoice times.
	firstLine := newGatheringLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	secondLine := newGatheringLineForLineEngineTest("line-2", billing.LineEngineTypeInvoice, false)
	secondLine.InvoiceAt = firstLine.InvoiceAt.Add(time.Hour)
	invoice := billing.GatheringInvoice{
		GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "invoice-1",
			},
		},
		Lines: billing.NewGatheringInvoiceLines(billing.GatheringLines{firstLine, secondLine}),
	}

	// When the updated gathering invoice is checked for billability.
	err := svc.checkIfGatheringLinesAreInvoicable(t.Context(), invoice, true)

	// Then each line is evaluated separately at its own InvoiceAt.
	require.NoError(t, err)
	require.Len(t, engine.inputs, 2)
	require.Equal(t, firstLine.InvoiceAt, engine.inputs[0].AsOf)
	require.Equal(t, billing.GatheringLines{firstLine}, engine.inputs[0].Lines)
	require.Equal(t, secondLine.InvoiceAt, engine.inputs[1].AsOf)
	require.Equal(t, billing.GatheringLines{secondLine}, engine.inputs[1].Lines)
}

func TestAreGatheringLinesBillableAsOfRequiresLinesFromInvoice(t *testing.T) {
	invoice := billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "ns"},
			ID:              "invoice-1",
		},
	}}

	tests := []struct {
		name       string
		mutateLine func(*billing.GatheringLine)
		wantError  string
	}{
		{
			name: "namespace mismatch",
			mutateLine: func(line *billing.GatheringLine) {
				line.Namespace = "other"
			},
			wantError: "namespace other does not match invoice namespace ns",
		},
		{
			name: "invoice mismatch",
			mutateLine: func(line *billing.GatheringLine) {
				line.InvoiceID = "other"
			},
			wantError: "invoice ID other does not match invoice ID invoice-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given a valid gathering line whose invoice ownership is changed.
			line := newGatheringLineForLineEngineTest("line", billing.LineEngineTypeInvoice, false)
			test.mutateLine(&line)

			// When billing prepares the line-engine billability batch.
			_, err := (&Service{}).areGatheringLinesBillableAsOf(t.Context(), billing.AreLinesBillableAsOfInput{
				Invoice: invoice,
				AsOf:    line.InvoiceAt,
				Lines:   billing.GatheringLines{line},
			})

			// Then the line is rejected before engine dispatch.
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

type billabilityLineEngine struct {
	billingtestutils.NoopLineEngine
	inputs []billing.AreLinesBillableAsOfInput
}

func (e *billabilityLineEngine) AreLinesBillableAsOf(_ context.Context, input billing.AreLinesBillableAsOfInput) ([]billing.IsLineBillableAsOfResult, error) {
	e.inputs = append(e.inputs, input)

	return lo.Map(input.Lines, func(billing.GatheringLine, int) billing.IsLineBillableAsOfResult {
		return billing.IsLineBillableAsOfResult{
			Billable: true,
			BillablePeriod: timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}
	}), nil
}

func TestDispatchSystemStandardLineDeletionsGroupsLinesByEngine(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}
	chargeEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeChargeUsageBased,
		},
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
	}

	invoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true)
	chargeLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeChargeUsageBased, true)

	require.NoError(t, svc.dispatchSystemStandardLineDeletions(t.Context(), invoice, []billing.GenericInvoiceLine{
		invoiceLine.AsGenericLine(),
		chargeLine.AsGenericLine(),
	}))

	require.Len(t, invoiceEngine.deletedBySystemInputs, 1)
	require.Equal(t, "invoice-1", invoiceEngine.deletedBySystemInputs[0].Invoice.ID)
	require.Equal(t, []string{"line-1"}, lineIDs(invoiceEngine.deletedBySystemInputs[0].Lines))

	require.Len(t, chargeEngine.deletedBySystemInputs, 1)
	require.Equal(t, "invoice-1", chargeEngine.deletedBySystemInputs[0].Invoice.ID)
	require.Equal(t, []string{"line-2"}, lineIDs(chargeEngine.deletedBySystemInputs[0].Lines))
}

func TestDispatchSystemStandardLineDeletionsReturnsEngineError(t *testing.T) {
	errEngineFailed := errors.New("engine failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		deletedBySystemErr: errEngineFailed,
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
	}

	err := svc.dispatchSystemStandardLineDeletions(t.Context(), invoice, []billing.GenericInvoiceLine{
		newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true).AsGenericLine(),
	})

	require.ErrorIs(t, err, errEngineFailed)
}

func TestOnUnsupportedCreditNoteGroupsLinesByEngine(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}
	chargeEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeChargeUsageBased,
		},
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
	}

	invoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true)
	chargeLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeChargeUsageBased, true)

	require.NoError(t, svc.OnUnsupportedCreditNote(t.Context(), billing.OnUnsupportedCreditNoteInput{
		Invoice: invoice,
		Lines: billing.StandardLines{
			invoiceLine,
			chargeLine,
		},
	}))

	require.Len(t, invoiceEngine.unsupportedCreditNoteInputs, 1)
	require.Equal(t, "invoice-1", invoiceEngine.unsupportedCreditNoteInputs[0].Invoice.ID)
	require.Equal(t, []string{"line-1"}, lineIDs(invoiceEngine.unsupportedCreditNoteInputs[0].Lines))

	require.Len(t, chargeEngine.unsupportedCreditNoteInputs, 1)
	require.Equal(t, "invoice-1", chargeEngine.unsupportedCreditNoteInputs[0].Invoice.ID)
	require.Equal(t, []string{"line-2"}, lineIDs(chargeEngine.unsupportedCreditNoteInputs[0].Lines))
}

func TestOnUnsupportedCreditNoteReturnsEngineError(t *testing.T) {
	errEngineFailed := errors.New("engine failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		unsupportedCreditNoteErr: errEngineFailed,
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	err := svc.OnUnsupportedCreditNote(t.Context(), billing.OnUnsupportedCreditNoteInput{
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
		},
		Lines: billing.StandardLines{
			newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true),
		},
	})

	require.ErrorIs(t, err, errEngineFailed)
}

func TestDeleteInvoiceSystemDeletionSourceDispatchesOnlyNonDeletedLines(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}
	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	activeLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	deletedLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeInvoice, true)

	sm := &InvoiceStateMachine{
		Service: svc,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
			Lines: billing.NewStandardInvoiceLines(billing.StandardLines{
				activeLine,
				deletedLine,
			}),
		},
	}

	require.NoError(t, sm.deleteInvoice(t.Context(), billing.DeleteInvoiceTriggerInput{
		Source: billing.ChangeSourceSystem,
	}))

	require.NotNil(t, sm.Invoice.DeletedAt)
	require.Equal(t, billing.ChangeSourceSystem, sm.Invoice.DeletionSource)
	require.Len(t, invoiceEngine.deletedBySystemInputs, 1)
	require.Equal(t, []string{"line-1"}, lineIDs(invoiceEngine.deletedBySystemInputs[0].Lines))
}

func TestDeleteInvoiceAPIRequestDispatchesNonDeletedLinesToAPIEditHook(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}
	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	activeLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	deletedLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeInvoice, true)

	sm := &InvoiceStateMachine{
		Service: svc,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
			Lines: billing.NewStandardInvoiceLines(billing.StandardLines{
				activeLine,
				deletedLine,
			}),
		},
	}

	require.NoError(t, sm.deleteInvoice(t.Context(), billing.DeleteInvoiceTriggerInput{
		Source: billing.ChangeSourceAPIRequest,
	}))

	require.NotNil(t, sm.Invoice.DeletedAt)
	require.Equal(t, billing.ChangeSourceAPIRequest, sm.Invoice.DeletionSource)
	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Equal(t, "invoice-1", invoiceEngine.apiEditInputs[0].Invoice.GetID())
	require.Equal(t, []string{"line-1"}, genericLineIDs(invoiceEngine.apiEditInputs[0].Deleted))
	require.Empty(t, invoiceEngine.deletedBySystemInputs)
}

func TestDeleteInvoiceAPIRequestReturnsChargeManagedLineErrorBeforeDeletingInvoice(t *testing.T) {
	chargeEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeChargeUsageBased,
		},
		changeErr: billing.ErrCannotUpdateChargeManagedLine,
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	sm := &InvoiceStateMachine{
		Service: svc,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
			Lines: billing.NewStandardInvoiceLines(billing.StandardLines{
				newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeChargeUsageBased, false),
			}),
		},
	}

	err := sm.deleteInvoice(t.Context(), billing.DeleteInvoiceTriggerInput{
		Source: billing.ChangeSourceAPIRequest,
	})

	require.ErrorIs(t, err, billing.ErrCannotUpdateChargeManagedLine)
	require.Nil(t, sm.Invoice.DeletedAt)
	require.Len(t, chargeEngine.apiEditInputs, 1)
	require.Equal(t, []string{"line-1"}, genericLineIDs(chargeEngine.apiEditInputs[0].Deleted))
	require.Empty(t, chargeEngine.deletedBySystemInputs)
}

func TestOnInvoiceFinalizingDoesNotApplyPartialEngineOutput(t *testing.T) {
	errFinalizationFailed := errors.New("finalization failed")
	finalizationCalls := 0
	onInvoiceFinalizing := func(_ context.Context, input billing.OnInvoiceFinalizingInput) (billing.StandardLines, error) {
		finalizationCalls++
		if finalizationCalls == 2 {
			return nil, errFinalizationFailed
		}

		input.Lines[0].Totals = totals.Totals{
			Amount: alpacadecimal.NewFromInt(100),
			Total:  alpacadecimal.NewFromInt(100),
		}

		return input.Lines, nil
	}

	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		onInvoiceFinalizing: onInvoiceFinalizing,
	}
	chargeEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeChargeUsageBased,
		},
		onInvoiceFinalizing: onInvoiceFinalizing,
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}
	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	lineTotals := totals.Totals{
		Amount: alpacadecimal.NewFromInt(10),
		Total:  alpacadecimal.NewFromInt(10),
	}
	invoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	invoiceLine.Totals = lineTotals
	chargeLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeChargeUsageBased, false)
	chargeLine.Totals = lineTotals

	sm := &InvoiceStateMachine{
		Service: svc,
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Namespace: "ns",
				ID:        "invoice-1",
			},
			Lines: billing.NewStandardInvoiceLines(billing.StandardLines{
				invoiceLine,
				chargeLine,
			}),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(20),
				Total:  alpacadecimal.NewFromInt(20),
			},
		},
	}

	// when: one engine returns updated totals before another engine fails
	err := sm.onInvoiceFinalizing(t.Context())

	// then: neither the successful engine output nor stale aggregate totals reach the invoice
	require.ErrorIs(t, err, errFinalizationFailed)
	require.Equal(t, 2, finalizationCalls)
	require.Equal(t, 10.0, sm.Invoice.Lines.GetByID("line-1").Totals.Total.InexactFloat64())
	require.Equal(t, 10.0, sm.Invoice.Lines.GetByID("line-2").Totals.Total.InexactFloat64())
	require.Equal(t, 20.0, sm.Invoice.Totals.Total.InexactFloat64())
}

func TestEngineRegistryAllowsSingleCreateLineRouter(t *testing.T) {
	registry := newEngineRegistry()
	router := staticCreateLineRouter{engine: billing.LineEngineTypeChargeFlatFee}

	require.NoError(t, registry.RegisterCreateLineRouter(router))
	require.ErrorContains(t, registry.RegisterCreateLineRouter(staticCreateLineRouter{engine: billing.LineEngineTypeChargeUsageBased}), "already registered")

	engine, err := registry.GetCreateLineRouter().GetLineEngineForCreateLine(newStandardLineForLineEngineTest("line-1", "", false))
	require.NoError(t, err)
	require.Equal(t, billing.LineEngineTypeChargeFlatFee, engine)
}

func TestDefaultCreateLineRouterReturnsInvoiceEngine(t *testing.T) {
	router := billing.DefaultCreateLineRouter{}

	engine, err := router.GetLineEngineForCreateLine(newStandardLineForLineEngineTest("line-1", "", false))
	require.NoError(t, err)
	require.Equal(t, billing.LineEngineTypeInvoice, engine)

	engine, err = router.GetLineEngineForCreateLine(newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeChargeFlatFee, false))
	require.NoError(t, err)
	require.Equal(t, billing.LineEngineTypeInvoice, engine)
}

type recordingLineEngine struct {
	billingtestutils.NoopLineEngine
	apiEditInputs               []billing.OnMutableInvoiceUpdateInput
	apiEditCreatedManagedBy     []billing.InvoiceLineManagedBy
	apiEditDeletedManagedBy     []billing.InvoiceLineManagedBy
	deletedBySystemInputs       []billing.OnMutableStandardLinesDeletedInput
	unsupportedCreditNoteInputs []billing.OnUnsupportedCreditNoteInput
	changeErr                   error
	deletedBySystemErr          error
	unsupportedCreditNoteErr    error
	onInvoiceFinalizing         func(context.Context, billing.OnInvoiceFinalizingInput) (billing.StandardLines, error)
}

type staticCreateLineRouter struct {
	engine billing.LineEngineType
}

func (r staticCreateLineRouter) GetLineEngineForCreateLine(billing.GenericInvoiceLineReader) (billing.LineEngineType, error) {
	return r.engine, nil
}

func (e *recordingLineEngine) ValidateMutableInvoiceLineEditViaAPI(_ context.Context, _ billing.OnMutableInvoiceUpdateInput) error {
	return nil
}

func (e *recordingLineEngine) OnMutableInvoiceLinesEditedViaAPI(_ context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	e.apiEditInputs = append(e.apiEditInputs, input)
	for _, line := range input.Created {
		e.apiEditCreatedManagedBy = append(e.apiEditCreatedManagedBy, line.GetManagedBy())
	}
	for _, line := range input.Deleted {
		e.apiEditDeletedManagedBy = append(e.apiEditDeletedManagedBy, line.GetManagedBy())
	}

	createdLines := slices.Clone(input.Created)

	updatedLines := make([]billing.GenericInvoiceLine, 0, len(input.Updated))
	for _, override := range input.Updated {
		line, err := override.ChangesToApply.Apply(override.ExistingLine)
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}

		updatedLines = append(updatedLines, line)
	}

	return billing.OnMutableInvoiceUpdateResult{
		CreatedLines: createdLines,
		UpdatedLines: updatedLines,
	}, e.changeErr
}

func (e *recordingLineEngine) OnMutableStandardLinesDeletedBySystem(_ context.Context, input billing.OnMutableStandardLinesDeletedInput) error {
	e.deletedBySystemInputs = append(e.deletedBySystemInputs, input)
	return e.deletedBySystemErr
}

func (e *recordingLineEngine) OnUnsupportedCreditNote(_ context.Context, input billing.OnUnsupportedCreditNoteInput) error {
	e.unsupportedCreditNoteInputs = append(e.unsupportedCreditNoteInputs, input)
	return e.unsupportedCreditNoteErr
}

func (e *recordingLineEngine) OnInvoiceFinalizing(ctx context.Context, input billing.OnInvoiceFinalizingInput) (billing.StandardLines, error) {
	if e.onInvoiceFinalizing == nil {
		return input.Lines, nil
	}

	return e.onInvoiceFinalizing(ctx, input)
}

func lineIDs(lines billing.StandardLines) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.ID)
	}

	return ids
}
