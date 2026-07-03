package billingservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func diffMutableInvoiceLinesForTest(before, after billing.GenericInvoiceReader, createLineRouter billing.CreateLineRouter) (mutableInvoiceLineDiff, error) {
	return diffMutableInvoiceLinesForTestWithSource(before, after, billing.ChangeSourceAPIRequest, createLineRouter)
}

func diffMutableInvoiceLinesForTestWithSource(
	before billing.GenericInvoiceReader,
	after billing.GenericInvoiceReader,
	source billing.ChangeSource,
	createLineRouter billing.CreateLineRouter,
) (mutableInvoiceLineDiff, error) {
	diff, err := diffMutableInvoiceLines(before, after, source, createLineRouter)
	if err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	diff.DefaultTaxCodeResolvers = noopDefaultTaxCodeResolvers()
	if err := diff.Validate(); err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	return diff, nil
}

func noopDefaultTaxCodeResolvers() billing.DefaultTaxCodeResolvers {
	return billing.DefaultTaxCodeResolvers{
		Invoicing: func(context.Context) (string, error) {
			return "", nil
		},
		CreditGrant: func(context.Context) (string, error) {
			return "", nil
		},
	}
}

const invoiceUpdateDefaultTaxCodeID = "default-tax-code-id"

func TestDiffInvoiceLinesByEngine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("newly-deleted", billing.LineEngineTypeInvoice, false),
			newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, true),
			newStandardLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false),
			newStandardLineForLineEngineTest("unchanged", billing.LineEngineTypeInvoice, false),
		}),
	}

	updated := newStandardLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false)
	updated.Name = "updated-name"

	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{
		newStandardLineForLineEngineTest("newly-deleted", billing.LineEngineTypeInvoice, true),
		updated,
		newStandardLineForLineEngineTest("unchanged", billing.LineEngineTypeInvoice, false),
		newStandardLineForLineEngineTest("created", billing.LineEngineTypeInvoice, false),
	})

	lineDiff, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	changesByEngine, err := lineDiff.GroupByLineEngine()
	require.NoError(t, err)
	changes := changesByEngine[billing.LineEngineTypeInvoice]
	require.Equal(t, []string{"created"}, genericLineIDs(changes.Created))
	require.Equal(t, []string{"updated"}, genericLineIDs(changes.Updated.Lines()))
	require.Equal(t, []string{"newly-deleted"}, genericLineIDs(changes.Deleted))
	require.NotNil(t, changes.Deleted[0].GetDeletedAt())
	require.Equal(t, "updated", changes.Updated[0].ExistingLine.GetID())
	require.Equal(t, "updated-name", changes.Updated[0].ChangesToApply.Name.OrEmpty())
}

func TestDiffInvoiceLinesByEngineSetsCreatedLineEngineFromRouter(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines(nil),
	}

	createdLine := newStandardLineForLineEngineTest("created", "", false)
	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{createdLine})

	lineDiff, err := diffMutableInvoiceLinesForTest(before, after, staticCreateLineRouter{engine: billing.LineEngineTypeChargeFlatFee})
	require.NoError(t, err)
	changesByEngine, err := lineDiff.GroupByLineEngine()
	require.NoError(t, err)

	changes := changesByEngine[billing.LineEngineTypeChargeFlatFee]
	require.Equal(t, billing.LineEngineTypeChargeFlatFee, createdLine.Engine)
	require.Equal(t, []string{"created"}, genericLineIDs(changes.Created))
}

func TestDiffInvoiceLinesByEngineIgnoresCreatedDeletedLine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines(nil),
	}

	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{
		newStandardLineForLineEngineTest("new-deleted-line", billing.LineEngineTypeInvoice, true),
	})

	lineDiff, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
}

func TestDiffInvoiceLinesByEngineIgnoresAlreadyDeletedLine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, true),
		}),
	}

	after := before

	lineDiff, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
	require.Empty(t, lineDiff.Created)
	require.Empty(t, lineDiff.Updated)
	require.Empty(t, lineDiff.Deleted)
}

func TestDiffInvoiceLinesByEngineReturnsErrorForRestoredDeletedLine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, true),
		}),
	}

	restoredLine := newStandardLineForLineEngineTest("already-deleted", billing.LineEngineTypeInvoice, false)
	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{restoredLine})

	_, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.ErrorContains(t, err, "line[already-deleted]: cannot restore a deleted line")
}

func TestDiffInvoiceLinesByEngineReturnsErrorForDeletedLineWithoutEngine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("deleted", "", false),
		}),
	}

	after := before
	after.Lines = billing.NewStandardInvoiceLines(nil)

	lineDiff, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	_, err = lineDiff.GroupByLineEngine()
	require.ErrorContains(t, err, "line[deleted]: line engine is required for deleted line")
}

func TestDiffInvoiceLinesByEngineReturnsErrorForUpdatedLineWithoutEngine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("updated", "", false),
		}),
	}

	updated := newStandardLineForLineEngineTest("updated", "", false)
	updated.Name = "updated-name"

	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{updated})

	_, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.ErrorContains(t, err, "line[updated]: line engine is required for updated line")
}

func TestDiffInvoiceLinesByEngineReturnsErrorForChangedLineEngine(t *testing.T) {
	before := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			newStandardLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false),
		}),
	}

	updated := newStandardLineForLineEngineTest("updated", billing.LineEngineTypeChargeUsageBased, false)
	updated.Name = "updated-name"

	after := before
	after.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{updated})

	_, err := diffMutableInvoiceLinesForTest(before, after, billing.DefaultCreateLineRouter{})
	require.ErrorContains(t, err, "line[updated]: line engine cannot be changed")
}

func TestDiffMutableInvoiceLinesSanitizesNilTaxConfigToDefaultNoDiff(t *testing.T) {
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(nil, nil)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())

	require.Nil(t, invoice.Lines.OrEmpty()[0].TaxConfig)
	require.Nil(t, edited.Lines.OrEmpty()[0].TaxConfig)

	sanitizedInvoice, err := svc.invoiceWithSanitizedTaxConfigForDiff(
		t.Context(),
		svc.defaultTaxCodeResolversForInvoiceUpdate(&invoice),
		&invoice,
	)
	require.NoError(t, err)
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, *sanitizedInvoice.GetGenericLines().OrEmpty()[0].GetTaxConfig().TaxCodeID)
}

func TestDiffMutableInvoiceLinesKeepsExplicitTaxCodeToDefaultDiff(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior:  &taxBehavior,
				TaxCodeID: lo.ToPtr(explicitTaxCodeID),
			},
		},
		nil,
	)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.Len(t, lineDiff.Updated, 1)

	updatedTaxConfig, ok := lineDiff.Updated[0].ChangesToApply.TaxConfig.Get()
	require.True(t, ok)
	require.NotNil(t, updatedTaxConfig)
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, *updatedTaxConfig.TaxCodeID)
}

func TestDiffMutableInvoiceLinesResolvesProviderDefaultTaxCodeIDMatchNoDiff(t *testing.T) {
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
			},
		},
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{},
			},
		},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())

	require.Nil(t, edited.Lines.OrEmpty()[0].TaxConfig.TaxCodeID)
	require.NotNil(t, edited.Lines.OrEmpty()[0].TaxConfig.Stripe)
	require.Empty(t, edited.Lines.OrEmpty()[0].TaxConfig.Stripe.Code)
}

func TestDiffMutableInvoiceLinesResolvedExplicitTaxCodeIDMatchNoDiff(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoiceTaxConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &taxBehavior,
			TaxCodeID: lo.ToPtr(explicitTaxCodeID),
		},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{
				Namespace: "ns",
				ID:        explicitTaxCodeID,
			},
			Key:  "explicit",
			Name: "Explicit Tax Code",
		},
	}
	editedTaxConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &taxBehavior,
			TaxCodeID: lo.ToPtr(explicitTaxCodeID),
		},
	}
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(invoiceTaxConfig, editedTaxConfig)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
}

func TestDiffMutableInvoiceLinesSystemSourceUsesFullTaxConfigEquality(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoiceTaxConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &taxBehavior,
			TaxCodeID: lo.ToPtr(explicitTaxCodeID),
		},
		TaxCode: &taxcode.TaxCode{
			NamespacedID: models.NamespacedID{
				Namespace: "ns",
				ID:        explicitTaxCodeID,
			},
			Key:  "explicit",
			Name: "Explicit Tax Code",
		},
	}
	editedTaxConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Behavior:  &taxBehavior,
			TaxCodeID: lo.ToPtr(explicitTaxCodeID),
		},
	}
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(invoiceTaxConfig, editedTaxConfig)

	apiLineDiff, err := diffMutableInvoiceLinesForTestWithSource(&invoice, &edited, billing.ChangeSourceAPIRequest, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	require.True(t, apiLineDiff.IsEmpty())

	systemLineDiff, err := diffMutableInvoiceLinesForTestWithSource(&invoice, &edited, billing.ChangeSourceSystem, billing.DefaultCreateLineRouter{})
	require.NoError(t, err)
	require.Len(t, systemLineDiff.Updated, 1)

	updatedTaxConfig, ok := systemLineDiff.Updated[0].ChangesToApply.TaxConfig.Get()
	require.True(t, ok)
	require.Equal(t, editedTaxConfig, updatedTaxConfig)
}

func TestDiffMutableInvoiceLinesResolvesProviderTaxCodeIDMatchNoDiff(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	stripeCode := "txcd_10000000"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior:  &taxBehavior,
				TaxCodeID: lo.ToPtr(explicitTaxCodeID),
			},
		},
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior: &taxBehavior,
				Stripe:   &productcatalog.StripeTaxConfig{Code: stripeCode},
			},
		},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[explicitTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{
			Namespace: "ns",
			ID:        explicitTaxCodeID,
		},
		Key:  "explicit",
		Name: "Explicit Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: stripeCode},
		},
	}

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
}

func TestDiffMutableInvoiceLinesBehaviorOnlyDifferenceProducesTaxConfigDiff(t *testing.T) {
	exclusiveBehavior := productcatalog.ExclusiveTaxBehavior
	inclusiveBehavior := productcatalog.InclusiveTaxBehavior
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior:  &exclusiveBehavior,
				TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
			},
		},
		&billing.TaxConfig{
			TaxConfig: productcatalog.TaxConfig{
				Behavior:  &inclusiveBehavior,
				TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
			},
		},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.Len(t, lineDiff.Updated, 1)

	updatedTaxConfig, ok := lineDiff.Updated[0].ChangesToApply.TaxConfig.Get()
	require.True(t, ok)
	require.NotNil(t, updatedTaxConfig)
	require.Equal(t, inclusiveBehavior, *updatedTaxConfig.Behavior)
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, *updatedTaxConfig.TaxCodeID)
}

func TestInvoiceWithSanitizedTaxConfigForDiffReturnsValidationErrorWhenTaxCodeIDCannotBeResolved(t *testing.T) {
	invoice, _ := standardInvoicePairForTaxConfigDiffTest(nil, nil)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService = &invoiceUpdateTaxCodeService{
		taxCodes: map[string]taxcode.TaxCode{
			"empty-id-provider-default-tax-code": {
				NamespacedID: models.NamespacedID{
					Namespace: "ns",
				},
				Key: taxcode.ProviderDefaultTaxCodeKey,
			},
		},
	}

	_, err := svc.invoiceWithSanitizedTaxConfigForDiff(t.Context(), billing.DefaultTaxCodeResolvers{
		Invoicing: func(context.Context) (string, error) {
			return "", nil
		},
		CreditGrant: func(context.Context) (string, error) {
			return "", nil
		},
	}, &invoice)
	require.ErrorContains(t, err, "validation error: cannot resolve tax code id")

	var validationErr billing.ValidationError
	require.ErrorAs(t, err, &validationErr)
}

func TestWithLineEngineInvoiceLineChangesGroupsAPIEditsByEngine(t *testing.T) {
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
		adapter:     preallocatingInvoiceLineAdapter{},
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))
	require.NoError(t, svc.RegisterLineEngine(chargeEngine))

	invoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	chargeLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeChargeUsageBased, false)
	createdInvoiceLine := newStandardLineForLineEngineTest("line-3", "", false)

	updatedInvoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	updatedInvoiceLine.Name = "edited-invoice-line"
	updatedChargeLine := newStandardLineForLineEngineTest("line-2", billing.LineEngineTypeChargeUsageBased, false)
	updatedChargeLine.Name = "edited-charge-line"

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace:   "ns",
			ID:          "invoice-1",
			SchemaLevel: 2,
		},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{invoiceLine, chargeLine}),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{updatedInvoiceLine, updatedChargeLine, createdInvoiceLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	editedInvoice, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)
	editedStandardInvoice, err := editedInvoice.AsInvoice().AsStandardInvoice()
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"line-1", "line-2", "line-3"}, lineIDs(editedStandardInvoice.Lines.OrEmpty()))

	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Equal(t, []string{"line-3"}, genericLineIDs(invoiceEngine.apiEditInputs[0].Created))
	require.Equal(t, []string{"line-1"}, genericLineIDs(invoiceEngine.apiEditInputs[0].Updated.Lines()))
	require.Empty(t, invoiceEngine.apiEditInputs[0].Deleted)

	require.Len(t, chargeEngine.apiEditInputs, 1)
	require.Empty(t, chargeEngine.apiEditInputs[0].Created)
	require.Equal(t, []string{"line-2"}, genericLineIDs(chargeEngine.apiEditInputs[0].Updated.Lines()))
	require.Empty(t, chargeEngine.apiEditInputs[0].Deleted)
}

func TestWithLineEngineInvoiceLineChangesReturnsEngineError(t *testing.T) {
	errEngineFailed := errors.New("engine failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		changeErr: errEngineFailed,
	}

	svc := &Service{
		adapter:     preallocatingInvoiceLineAdapter{},
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	invoiceLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	updatedLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	updatedLine.Name = "edited-invoice-line"

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{invoiceLine}),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{updatedLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	_, err = svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.ErrorContains(t, err, errEngineFailed.Error())
}

func TestWithLineEngineInvoiceLineChangesPreallocatesCreatedLineID(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		adapter:     preallocatingInvoiceLineAdapter{},
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	createdLine := newStandardLineForLineEngineTest("", "", false)
	createdLine.Name = "created"

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace:   "ns",
			ID:          "invoice-1",
			Currency:    "USD",
			SchemaLevel: 2,
		},
		Lines: billing.NewStandardInvoiceLines(nil),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{createdLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	editedInvoice, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)
	editedStandardInvoice, err := editedInvoice.AsInvoice().AsStandardInvoice()
	require.NoError(t, err)

	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Len(t, invoiceEngine.apiEditInputs[0].Created, 1)
	createdInputLine := invoiceEngine.apiEditInputs[0].Created[0]
	require.NotEmpty(t, createdInputLine.GetID())
	require.Equal(t, "invoice-1", createdInputLine.GetInvoiceID())
	require.Equal(t, billing.LineEngineTypeInvoice, createdInputLine.GetEngine())
	require.Equal(t, createdInputLine.GetID(), editedStandardInvoice.Lines.OrEmpty()[0].ID)
}

func TestApplyManualInvoiceLineOverridesMarksManualChanges(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		adapter:     preallocatingInvoiceLineAdapter{},
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	originalLine := newStandardLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false)
	originalLine.ManagedBy = billing.SystemManagedLine

	updatedLine := newStandardLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false)
	updatedLine.ManagedBy = billing.SystemManagedLine
	updatedLine.Name = "updated-name"

	createdLine := newStandardLineForLineEngineTest("created", billing.LineEngineTypeInvoice, false)
	createdLine.ManagedBy = billing.SystemManagedLine

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace:   "ns",
			ID:          "invoice-1",
			Status:      billing.StandardInvoiceStatusGathering,
			SchemaLevel: 2,
		},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{originalLine}),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{updatedLine, createdLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	editedInvoice, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)
	editedStandardInvoice, err := editedInvoice.AsInvoice().AsStandardInvoice()
	require.NoError(t, err)

	resultLines := editedStandardInvoice.Lines.OrEmpty()
	require.Len(t, resultLines, 2)
	require.Equal(t, billing.ManuallyManagedLine, resultLines[0].ManagedBy)
	require.Equal(t, billing.ManuallyManagedLine, resultLines[1].ManagedBy)

	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Equal(t, billing.SystemManagedLine, invoiceEngine.apiEditInputs[0].Updated[0].ExistingLine.GetManagedBy())
	require.Equal(t, []billing.InvoiceLineManagedBy{billing.ManuallyManagedLine}, invoiceEngine.apiEditCreatedManagedBy)
}

func TestApplyManualInvoiceLineOverridesMarksManualDeletes(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	originalLine := newStandardLineForLineEngineTest("deleted", billing.LineEngineTypeInvoice, false)
	originalLine.ManagedBy = billing.SystemManagedLine

	deletedLine := newStandardLineForLineEngineTest("deleted", billing.LineEngineTypeInvoice, true)
	deletedLine.ManagedBy = billing.SystemManagedLine

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{originalLine}),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{deletedLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	editedInvoice, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)
	editedStandardInvoice, err := editedInvoice.AsInvoice().AsStandardInvoice()
	require.NoError(t, err)

	resultLines := editedStandardInvoice.Lines.OrEmpty()
	require.Len(t, resultLines, 1)
	require.NotNil(t, resultLines[0].DeletedAt)
	require.Equal(t, billing.ManuallyManagedLine, resultLines[0].ManagedBy)

	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Equal(t, []billing.InvoiceLineManagedBy{billing.SystemManagedLine}, invoiceEngine.apiEditDeletedManagedBy)
}

func TestApplyManualInvoiceLineOverridesMarksGatheringManualChanges(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := &Service{
		adapter:     preallocatingInvoiceLineAdapter{},
		lineEngines: newEngineRegistry(),
	}

	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	originalLine := newGatheringLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false)
	originalLine.ManagedBy = billing.SystemManagedLine

	updatedLine := newGatheringLineForLineEngineTest("updated", billing.LineEngineTypeInvoice, false)
	updatedLine.ManagedBy = billing.SystemManagedLine
	updatedLine.Name = "updated-name"

	createdLine := newGatheringLineForLineEngineTest("created", billing.LineEngineTypeInvoice, false)
	createdLine.ManagedBy = billing.SystemManagedLine

	invoice := billing.GatheringInvoice{
		GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "invoice-1",
				Name:            "invoice-1",
			},
			Currency:      currencyx.Code("USD"),
			ServicePeriod: originalLine.ServicePeriod,
			SchemaLevel:   1,
		},
		Lines: billing.NewGatheringInvoiceLines(billing.GatheringLines{originalLine}),
	}

	edited := invoice
	edited.Lines = billing.NewGatheringInvoiceLines(billing.GatheringLines{updatedLine, createdLine})

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	editedInvoice, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)
	editedGatheringInvoice, err := editedInvoice.AsInvoice().AsGatheringInvoice()
	require.NoError(t, err)

	resultLines := editedGatheringInvoice.Lines.OrEmpty()
	require.Len(t, resultLines, 2)
	require.Equal(t, billing.ManuallyManagedLine, resultLines[0].ManagedBy)
	require.Equal(t, billing.ManuallyManagedLine, resultLines[1].ManagedBy)

	require.Len(t, invoiceEngine.apiEditInputs, 1)
	require.Equal(t, billing.SystemManagedLine, invoiceEngine.apiEditInputs[0].Updated[0].ExistingLine.GetManagedBy())
	require.Equal(t, []billing.InvoiceLineManagedBy{billing.ManuallyManagedLine}, invoiceEngine.apiEditCreatedManagedBy)
}

func newStandardLineForLineEngineTest(id string, engine billing.LineEngineType, deleted bool) *billing.StandardLine {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var deletedAt *time.Time
	if deleted {
		deletedAt = lo.ToPtr(now.Add(time.Hour))
	}

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel: models.ManagedModel{
					DeletedAt: deletedAt,
				},
				ID:   id,
				Name: id,
			},
			Engine:    engine,
			InvoiceID: "invoice-1",
			Currency:  currencyx.Code("USD"),
			ManagedBy: billing.ManuallyManagedLine,
			Period: timeutil.ClosedPeriod{
				From: now,
				To:   now.Add(time.Hour),
			},
			InvoiceAt: now.Add(time.Hour),
			TaxConfig: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
				},
			},
		},
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(100),
			}),
		},
	}
}

func newGatheringLineForLineEngineTest(id string, engine billing.LineEngineType, deleted bool) billing.GatheringLine {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var deletedAt *time.Time
	if deleted {
		deletedAt = lo.ToPtr(now.Add(time.Hour))
	}

	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel: models.ManagedModel{
					DeletedAt: deletedAt,
				},
				ID:   id,
				Name: id,
			},
			Engine:    engine,
			InvoiceID: "invoice-1",
			Currency:  currencyx.Code("USD"),
			ManagedBy: billing.ManuallyManagedLine,
			ServicePeriod: timeutil.ClosedPeriod{
				From: now,
				To:   now.Add(time.Hour),
			},
			InvoiceAt: now.Add(time.Hour),
			TaxConfig: &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
			},
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(100),
			}),
		},
	}
}

func genericLineIDs(lines []billing.GenericInvoiceLine) []string {
	return lo.Map(lines, func(line billing.GenericInvoiceLine, _ int) string {
		return line.GetID()
	})
}

func standardInvoicePairForTaxConfigDiffTest(beforeTaxConfig, afterTaxConfig *billing.TaxConfig) (billing.StandardInvoice, billing.StandardInvoice) {
	beforeLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	beforeLine.TaxConfig = cloneBillingTaxConfigForTest(beforeTaxConfig)

	afterLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false)
	afterLine.TaxConfig = cloneBillingTaxConfigForTest(afterTaxConfig)

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace:   "ns",
			ID:          "invoice-1",
			SchemaLevel: 2,
			Workflow: billing.InvoiceWorkflow{
				Config: billing.WorkflowConfig{
					Invoicing: billing.InvoicingConfig{
						DefaultTaxConfig: &productcatalog.TaxConfig{
							TaxCodeID: lo.ToPtr(invoiceUpdateDefaultTaxCodeID),
						},
					},
				},
			},
		},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{beforeLine}),
	}

	edited := invoice
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{afterLine})

	return invoice, edited
}

func cloneBillingTaxConfigForTest(taxConfig *billing.TaxConfig) *billing.TaxConfig {
	if taxConfig == nil {
		return nil
	}

	cloned := taxConfig.Clone()
	return &cloned
}

func serviceForInvoiceTaxConfigDiffTest() *Service {
	return &Service{
		lineEngines: newEngineRegistry(),
		taxCodeService: &invoiceUpdateTaxCodeService{
			taxCodes: map[string]taxcode.TaxCode{
				invoiceUpdateDefaultTaxCodeID: {
					NamespacedID: models.NamespacedID{
						Namespace: "ns",
						ID:        invoiceUpdateDefaultTaxCodeID,
					},
					Key:  taxcode.ProviderDefaultTaxCodeKey,
					Name: "Default Tax Code",
				},
			},
		},
	}
}

var _ taxcode.Service = (*invoiceUpdateTaxCodeService)(nil)

type invoiceUpdateTaxCodeService struct {
	taxCodes map[string]taxcode.TaxCode
}

func (s *invoiceUpdateTaxCodeService) CreateTaxCode(context.Context, taxcode.CreateTaxCodeInput) (taxcode.TaxCode, error) {
	return taxcode.TaxCode{}, errors.New("CreateTaxCode is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) UpdateTaxCode(context.Context, taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	return taxcode.TaxCode{}, errors.New("UpdateTaxCode is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) ListTaxCodes(context.Context, taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	return pagination.Result[taxcode.TaxCode]{}, errors.New("ListTaxCodes is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) GetTaxCode(_ context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	tc, ok := s.taxCodes[input.ID]
	if !ok || tc.Namespace != input.Namespace {
		return taxcode.TaxCode{}, taxcode.NewTaxCodeNotFoundError(input.ID)
	}

	return tc, nil
}

func (s *invoiceUpdateTaxCodeService) GetTaxCodeByKey(_ context.Context, input taxcode.GetTaxCodeByKeyInput) (taxcode.TaxCode, error) {
	for _, tc := range s.taxCodes {
		if tc.Namespace == input.Namespace && tc.Key == input.Key {
			return tc, nil
		}
	}

	return taxcode.TaxCode{}, taxcode.NewTaxCodeByKeyNotFoundError(input.Key)
}

func (s *invoiceUpdateTaxCodeService) GetTaxCodeByAppMapping(context.Context, taxcode.GetTaxCodeByAppMappingInput) (taxcode.TaxCode, error) {
	return taxcode.TaxCode{}, errors.New("GetTaxCodeByAppMapping is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) GetOrCreateByAppMapping(_ context.Context, input taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error) {
	for _, tc := range s.taxCodes {
		mapping, ok := tc.GetAppMapping(input.AppType)
		if ok && mapping.TaxCode == input.TaxCode {
			return tc, nil
		}
	}

	return taxcode.TaxCode{}, taxcode.NewTaxCodeByAppMappingNotFoundError(string(input.AppType), input.TaxCode)
}

func (s *invoiceUpdateTaxCodeService) DeleteTaxCode(context.Context, taxcode.DeleteTaxCodeInput) error {
	return errors.New("DeleteTaxCode is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) GetOrganizationDefaultTaxCodes(context.Context, taxcode.GetOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	return taxcode.OrganizationDefaultTaxCodes{}, errors.New("GetOrganizationDefaultTaxCodes is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) UpsertOrganizationDefaultTaxCodes(context.Context, taxcode.UpsertOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	return taxcode.OrganizationDefaultTaxCodes{}, errors.New("UpsertOrganizationDefaultTaxCodes is not supported in this test")
}

func (s *invoiceUpdateTaxCodeService) RegisterHooks(...models.ServiceHook[taxcode.TaxCode]) {}

type preallocatingInvoiceLineAdapter struct {
	billing.Adapter
}

func (preallocatingInvoiceLineAdapter) UpsertInvoiceLines(_ context.Context, input billing.UpsertInvoiceLinesAdapterInput) ([]*billing.StandardLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return input.Lines.Clone()
}

func (preallocatingInvoiceLineAdapter) UpdateGatheringInvoice(_ context.Context, input billing.UpdateGatheringInvoiceAdapterInput) error {
	return input.Validate()
}
