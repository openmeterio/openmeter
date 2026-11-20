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
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/taxcoderesolver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, sanitizedInvoice.GetGenericLines().OrEmpty()[0].GetTaxConfig().TaxCodeID)
}

func TestDiffMutableInvoiceLinesKeepsExplicitTaxCodeToDefaultDiff(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&productcatalog.TaxCodeConfig{
			Behavior:  &taxBehavior,
			TaxCodeID: explicitTaxCodeID,
		},
		nil,
	)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[explicitTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{
			Namespace: "ns",
			ID:        explicitTaxCodeID,
		},
		Key:  "explicit",
		Name: "Explicit Tax Code",
	}

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.Len(t, lineDiff.Updated, 1)

	updatedTaxConfig, ok := lineDiff.Updated[0].ChangesToApply.TaxConfig.Get()
	require.True(t, ok)
	require.NotNil(t, updatedTaxConfig)
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, updatedTaxConfig.TaxCodeID)
}

func TestDiffMutableInvoiceLinesEmptyTaxConfigMatchesProviderDefaultNoDiff(t *testing.T) {
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&productcatalog.TaxCodeConfig{
			TaxCodeID: invoiceUpdateDefaultTaxCodeID,
		},
		&productcatalog.TaxCodeConfig{},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())

	require.Empty(t, edited.Lines.OrEmpty()[0].TaxConfig.TaxCodeID)
}

func TestDiffMutableInvoiceLinesResolvedExplicitTaxCodeIDMatchNoDiff(t *testing.T) {
	explicitTaxCodeID := "explicit-tax-code-id"
	taxBehavior := productcatalog.ExclusiveTaxBehavior
	invoiceTaxConfig := &productcatalog.TaxCodeConfig{
		Behavior:  &taxBehavior,
		TaxCodeID: explicitTaxCodeID,
	}
	editedTaxConfig := &productcatalog.TaxCodeConfig{
		Behavior:  &taxBehavior,
		TaxCodeID: explicitTaxCodeID,
	}
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(invoiceTaxConfig, editedTaxConfig)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[explicitTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: explicitTaxCodeID},
		Key:          "explicit",
		Name:         "Explicit Tax Code",
	}

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
}

func TestDiffMutableInvoiceLinesAllowsReplacingDeletedTaxCode(t *testing.T) {
	deletedTaxCodeID := "deleted-tax-code-id"
	replacementTaxCodeID := "replacement-tax-code-id"
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&productcatalog.TaxCodeConfig{
			TaxCodeID: deletedTaxCodeID,
		},
		&productcatalog.TaxCodeConfig{
			TaxCodeID: replacementTaxCodeID,
		},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()
	taxCodes := svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes
	taxCodes[deletedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: deletedTaxCodeID},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
		Key:          "deleted",
		Name:         "Deleted Tax Code",
	}
	taxCodes[replacementTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: replacementTaxCodeID},
		Key:          "replacement",
		Name:         "Replacement Tax Code",
	}

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)
	require.Len(t, lineDiff.Updated, 1)

	updatedTaxConfig, ok := lineDiff.Updated[0].ChangesToApply.TaxConfig.Get()
	require.True(t, ok)
	require.Equal(t, replacementTaxCodeID, updatedTaxConfig.TaxCodeID)
}

func TestDiffMutableInvoiceLinesAllowsUnchangedDeletedTaxCode(t *testing.T) {
	deletedTaxCodeID := "deleted-tax-code-id"
	taxConfig := &productcatalog.TaxCodeConfig{
		TaxCodeID: deletedTaxCodeID,
	}
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(taxConfig, taxConfig)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[deletedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: deletedTaxCodeID},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
		Key:          "deleted",
		Name:         "Deleted Tax Code",
	}

	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)

	// The edit leaves the line's Tax Code untouched, so the stored deleted reference
	// passes; only deleted IDs the edit newly sets are rejected.
	require.NoError(t, err)
	require.True(t, lineDiff.IsEmpty())
}

func TestDiffMutableInvoiceLinesRejectsNewlyReferencedDeletedTaxCode(t *testing.T) {
	unchangedTaxCodeID := "unchanged-tax-code-id"
	deletedTaxCodeID := "deleted-tax-code-id"
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&productcatalog.TaxCodeConfig{
			TaxCodeID: unchangedTaxCodeID,
		},
		&productcatalog.TaxCodeConfig{
			TaxCodeID: deletedTaxCodeID,
		},
	)
	svc := serviceForInvoiceTaxConfigDiffTest()
	taxCodes := svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes
	taxCodes[unchangedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: unchangedTaxCodeID},
		Key:          "unchanged",
		Name:         "Unchanged Tax Code",
	}
	taxCodes[deletedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: deletedTaxCodeID},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
		Key:          "deleted",
		Name:         "Deleted Tax Code",
	}

	_, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)

	require.ErrorIs(t, err, billing.ErrTaxCodeDeleted)
}

func TestDiffMutableInvoiceLinesReportsNewLineWithDeletedTaxCodeByIndex(t *testing.T) {
	deletedTaxCodeID := "deleted-tax-code-id"
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(nil, nil)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[deletedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: deletedTaxCodeID},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
		Key:          "deleted",
		Name:         "Deleted Tax Code",
	}

	// given: an edit that keeps line-1 and appends a new line, which has no ID yet
	newLine := newStandardLineForLineEngineTest("", billing.LineEngineTypeInvoice, false)
	newLine.TaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: deletedTaxCodeID}
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{edited.Lines.OrEmpty()[0], newLine})

	// when: diffing the edit
	_, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)

	// then: the new line is addressed by its position in the submitted lines
	require.ErrorIs(t, err, billing.ErrTaxCodeDeleted)
	require.ErrorContains(t, err, "lines/1/tax_config/tax_code")
	require.NotContains(t, err.Error(), "lines//")
}

func TestDiffMutableInvoiceLinesReportsEveryLineReferencingSameDeletedTaxCode(t *testing.T) {
	deletedTaxCodeID := "deleted-tax-code-id"
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(nil, nil)
	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.taxCodeService.(*invoiceUpdateTaxCodeService).taxCodes[deletedTaxCodeID] = taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: deletedTaxCodeID},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Now())},
		Key:          "deleted",
		Name:         "Deleted Tax Code",
	}

	// given: an edit where an existing line and a new line both newly reference the same deleted code
	deletedTaxConfig := &productcatalog.TaxCodeConfig{TaxCodeID: deletedTaxCodeID}
	changedLine := edited.Lines.OrEmpty()[0]
	changedLine.TaxConfig = cloneBillingTaxConfigForTest(deletedTaxConfig)
	newLine := newStandardLineForLineEngineTest("", billing.LineEngineTypeInvoice, false)
	newLine.TaxConfig = cloneBillingTaxConfigForTest(deletedTaxConfig)
	edited.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{changedLine, newLine})

	// when: diffing the edit
	_, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)

	// then: each referencing line gets its own issue
	require.ErrorIs(t, err, billing.ErrTaxCodeDeleted)
	require.ErrorContains(t, err, "lines/0/tax_config/tax_code")
	require.ErrorContains(t, err, "lines/1/tax_config/tax_code")
}

func TestDiffMutableInvoiceLinesBehaviorOnlyDifferenceProducesTaxConfigDiff(t *testing.T) {
	exclusiveBehavior := productcatalog.ExclusiveTaxBehavior
	inclusiveBehavior := productcatalog.InclusiveTaxBehavior
	invoice, edited := standardInvoicePairForTaxConfigDiffTest(
		&productcatalog.TaxCodeConfig{
			Behavior:  &exclusiveBehavior,
			TaxCodeID: invoiceUpdateDefaultTaxCodeID,
		},
		&productcatalog.TaxCodeConfig{
			Behavior:  &inclusiveBehavior,
			TaxCodeID: invoiceUpdateDefaultTaxCodeID,
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
	require.Equal(t, invoiceUpdateDefaultTaxCodeID, updatedTaxConfig.TaxCodeID)
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

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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

	issues, systemErr := billing.ToValidationIssues(err)
	require.Nil(t, issues)
	require.Equal(t, err, systemErr)
	require.ErrorIs(t, err, errEngineFailed)
}

func TestWithLineEngineInvoiceLineChangesReturnsEngineValidationIssue(t *testing.T) {
	engineIssue := billing.NewValidationError("engine_validation_failed", "engine validation failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		changeErr: engineIssue,
	}

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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
	require.ErrorIs(t, err, engineIssue)

	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{
		{
			Severity:  billing.ValidationIssueSeverityCritical,
			Code:      engineIssue.Code,
			Message:   engineIssue.Message,
			Component: billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice),
		},
	}, issues)
}

func TestWithLineEngineInvoiceLineChangesAcceptsWarningOutput(t *testing.T) {
	// Given an engine that returns a usable edit alongside a typed warning.
	warning := billing.NewValidationWarning("engine_warning", "engine warning")
	engine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice},
		changeErr:      warning,
	}
	svc := serviceForInvoiceTaxConfigDiffTest()
	require.NoError(t, svc.RegisterLineEngine(engine))
	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{Namespace: "ns", ID: "invoice-1"},
		Lines: billing.NewStandardInvoiceLines(billing.StandardLines{
			newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, false),
		}),
	}
	edited, err := invoice.Clone()
	require.NoError(t, err)
	edited.Lines.OrEmpty()[0].Name = "edited name"
	lineDiff, err := svc.diffMutableInvoiceLines(t.Context(), &invoice, &edited, billing.ChangeSourceAPIRequest)
	require.NoError(t, err)

	// When dispatch accepts warning-only output and checks its exact line IDs.
	result, err := svc.applyAPIInvoiceLineEdits(t.Context(), applyAPIInvoiceLineEditsInput{
		EditedInvoice: edited,
		LineDiff:      lineDiff,
	})
	require.NoError(t, err)

	// Then the changed line and owning issue are returned together.
	resultInvoice, err := result.AsInvoice().AsStandardInvoice()
	require.NoError(t, err)
	require.Equal(t, "edited name", resultInvoice.Lines.GetByID("line-1").Name)
	warning.Component = billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice)
	require.Equal(t, billing.ValidationIssues{warning}, resultInvoice.ValidationIssues)
}

func TestDispatchAPIStandardLineDeletionsRecordsEngineErrorsAsValidationIssues(t *testing.T) {
	errEngineFailed := errors.New("engine failed")
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
		changeErr: errEngineFailed,
	}

	svc := serviceForInvoiceTaxConfigDiffTest()
	require.NoError(t, svc.RegisterLineEngine(invoiceEngine))

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
	}
	deletedLine := newStandardLineForLineEngineTest("line-1", billing.LineEngineTypeInvoice, true)

	err := svc.dispatchAPIStandardLineDeletions(t.Context(), invoice, billing.StandardLines{deletedLine})
	require.ErrorIs(t, err, errEngineFailed)

	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{
		{
			Severity:  billing.ValidationIssueSeverityCritical,
			Message:   errEngineFailed.Error(),
			Component: billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice),
		},
	}, issues)
}

func TestWithLineEngineInvoiceLineChangesPreallocatesCreatedLineID(t *testing.T) {
	invoiceEngine := &recordingLineEngine{
		NoopLineEngine: billingtestutils.NoopLineEngine{
			EngineType: billing.LineEngineTypeInvoice,
		},
	}

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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

	svc := serviceForInvoiceTaxConfigDiffTest()

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

	svc := serviceForInvoiceTaxConfigDiffTest()
	svc.adapter = preallocatingInvoiceLineAdapter{}

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
			Currency:      currencyx.FiatCode("USD"),
			ServicePeriod: originalLine.ServicePeriod,
			SchemaLevel:   billing.CurrentInvoiceSchemaLevel,
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
			Currency:  currencyx.FiatCode("USD"),
			ManagedBy: billing.ManuallyManagedLine,
			Period: timeutil.ClosedPeriod{
				From: now,
				To:   now.Add(time.Hour),
			},
			InvoiceAt: now.Add(time.Hour),
			TaxConfig: &productcatalog.TaxCodeConfig{
				TaxCodeID: invoiceUpdateDefaultTaxCodeID,
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
			Currency:  currencyx.FiatCode("USD"),
			ManagedBy: billing.ManuallyManagedLine,
			ServicePeriod: timeutil.ClosedPeriod{
				From: now,
				To:   now.Add(time.Hour),
			},
			InvoiceAt: now.Add(time.Hour),
			TaxConfig: &productcatalog.TaxCodeConfig{
				TaxCodeID: invoiceUpdateDefaultTaxCodeID,
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

func standardInvoicePairForTaxConfigDiffTest(beforeTaxConfig, afterTaxConfig *productcatalog.TaxCodeConfig) (billing.StandardInvoice, billing.StandardInvoice) {
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
						DefaultTaxConfig: &productcatalog.TaxCodeConfig{
							TaxCodeID: invoiceUpdateDefaultTaxCodeID,
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

func cloneBillingTaxConfigForTest(taxConfig *productcatalog.TaxCodeConfig) *productcatalog.TaxCodeConfig {
	if taxConfig == nil {
		return nil
	}

	cloned := taxConfig.Clone()

	return &cloned
}

func serviceForInvoiceTaxConfigDiffTest() *Service {
	taxCodeService := &invoiceUpdateTaxCodeService{
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
	}

	taxCodeResolver := lo.Must(taxcoderesolver.New(taxCodeService))

	return &Service{
		adapter:         preallocatingInvoiceLineAdapter{},
		lineEngines:     newEngineRegistry(),
		taxCodeService:  taxCodeService,
		taxCodeResolver: taxCodeResolver,
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

func (s *invoiceUpdateTaxCodeService) ListTaxCodes(_ context.Context, input taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	if input.Page.PageNumber > 1 {
		return pagination.Result[taxcode.TaxCode]{}, nil
	}

	var items []taxcode.TaxCode

	for _, tc := range s.taxCodes {
		if tc.Namespace != input.Namespace {
			continue
		}

		if tc.DeletedAt != nil && !input.IncludeDeleted {
			continue
		}

		if len(input.IDs) > 0 && !slices.Contains(input.IDs, tc.ID) {
			continue
		}

		items = append(items, tc)
	}

	return pagination.Result[taxcode.TaxCode]{
		Items: items,
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   len(items),
		},
		TotalCount: len(items),
	}, nil
}

func (s *invoiceUpdateTaxCodeService) GetTaxCode(_ context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	tc, ok := s.taxCodes[input.ID]
	if !ok || tc.Namespace != input.Namespace || (tc.DeletedAt != nil && !input.IncludeDeleted) {
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

type preallocatingInvoiceLineAdapter struct {
	billing.Adapter
}

func (preallocatingInvoiceLineAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, &validationWarningTransactionDriver{}, nil
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
