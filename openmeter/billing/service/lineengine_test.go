package billingservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
)

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

	require.NoError(t, svc.dispatchSystemStandardLineDeletions(t.Context(), invoice, mutableInvoiceLineDiff{
		OnMutableInvoiceUpdateInput: billing.OnMutableInvoiceUpdateInput{
			Deleted: []billing.GenericInvoiceLine{
				invoiceLine.AsGenericLine(),
				chargeLine.AsGenericLine(),
			},
		},
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

	err := svc.dispatchSystemStandardLineDeletions(t.Context(), invoice, mutableInvoiceLineDiff{
		OnMutableInvoiceUpdateInput: billing.OnMutableInvoiceUpdateInput{
			Deleted: []billing.GenericInvoiceLine{
				newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true).AsGenericLine(),
			},
		},
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
	deletedBySystemInputs       []billing.OnMutableStandardLinesDeletedInput
	unsupportedCreditNoteInputs []billing.OnUnsupportedCreditNoteInput
	changeErr                   error
	deletedBySystemErr          error
	unsupportedCreditNoteErr    error
}

type staticCreateLineRouter struct {
	engine billing.LineEngineType
}

func (r staticCreateLineRouter) GetLineEngineForCreateLine(billing.GenericInvoiceLineReader) (billing.LineEngineType, error) {
	return r.engine, nil
}

func (e *recordingLineEngine) OnMutableInvoiceLinesEditedViaAPI(_ context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	e.apiEditInputs = append(e.apiEditInputs, input)
	createdLines := make([]billing.GenericInvoiceLine, 0, len(input.Created))
	for _, line := range input.Created {
		line.SetManagedBy(billing.ManuallyManagedLine)
		createdLines = append(createdLines, line)
	}

	updatedLines := make([]billing.GenericInvoiceLine, 0, len(input.Updated))
	for _, override := range input.Updated {
		line, err := override.ChangesToApply.Apply(override.ExistingLine)
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}

		line.SetManagedBy(billing.ManuallyManagedLine)
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

func lineIDs(lines billing.StandardLines) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.ID)
	}

	return ids
}

func lineWithHeaderIDs(lines billing.LinesWithInvoiceHeaders) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.Line.GetID())
	}

	return ids
}
