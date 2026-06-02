package billingservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
)

func TestOnMutableStandardLinesDeletedGroupsLinesByEngine(t *testing.T) {
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

	require.NoError(t, svc.OnMutableStandardLinesDeleted(t.Context(), billing.OnMutableStandardLinesDeletedInput{
		Invoice: invoice,
		Lines: billing.StandardLines{
			invoiceLine,
			chargeLine,
		},
	}))

	require.Len(t, invoiceEngine.inputs, 1)
	require.Equal(t, "invoice-1", invoiceEngine.inputs[0].Invoice.ID)
	require.Equal(t, []string{"line-1"}, lineIDs(invoiceEngine.inputs[0].Lines))

	require.Len(t, chargeEngine.inputs, 1)
	require.Equal(t, "invoice-1", chargeEngine.inputs[0].Invoice.ID)
	require.Equal(t, []string{"line-2"}, lineIDs(chargeEngine.inputs[0].Lines))
}

func TestOnMutableStandardLinesDeletedReturnsEngineError(t *testing.T) {
	errEngineFailed := errors.New("engine failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		err: errEngineFailed,
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	err := svc.OnMutableStandardLinesDeleted(t.Context(), billing.OnMutableStandardLinesDeletedInput{
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

type recordingLineEngine struct {
	billingtestutils.NoopLineEngine
	inputs                      []billing.OnMutableStandardLinesDeletedInput
	unsupportedCreditNoteInputs []billing.OnUnsupportedCreditNoteInput
	err                         error
	unsupportedCreditNoteErr    error
}

func (e *recordingLineEngine) OnMutableStandardLinesDeleted(_ context.Context, input billing.OnMutableStandardLinesDeletedInput) error {
	e.inputs = append(e.inputs, input)
	return e.err
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
