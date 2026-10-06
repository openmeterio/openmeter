package reconciler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type chargeReplacementGuardChargesServiceStub struct {
	charges.Service
	byID    map[string]charges.Charge
	err     error
	lookups []charges.GetByIDInput
}

func (s *chargeReplacementGuardChargesServiceStub) GetByID(_ context.Context, input charges.GetByIDInput) (charges.Charge, error) {
	s.lookups = append(s.lookups, input)
	if s.err != nil {
		return charges.Charge{}, s.err
	}

	charge, ok := s.byID[input.ChargeID.ID]
	if !ok {
		return charges.Charge{}, errors.New("charge not found")
	}

	return charge, nil
}

type chargeReplacementGuardBillingServiceStub struct {
	billing.Service
	byID    map[string]billing.StandardInvoice
	extra   []billing.StandardInvoice
	err     error
	lookups []billing.ListStandardInvoicesInput
}

func (s *chargeReplacementGuardBillingServiceStub) ListStandardInvoices(_ context.Context, input billing.ListStandardInvoicesInput) (billing.ListStandardInvoicesResponse, error) {
	s.lookups = append(s.lookups, input)
	if s.err != nil {
		return billing.ListStandardInvoicesResponse{}, s.err
	}

	response := billing.ListStandardInvoicesResponse{}
	for _, id := range input.IDs {
		if invoice, ok := s.byID[id]; ok {
			invoice.ID = id
			response.Items = append(response.Items, invoice)
		}
	}

	response.Items = append(response.Items, s.extra...)

	return response, nil
}

func newComparisonReconciler() *Service {
	return &Service{
		chargesService: &chargeReplacementGuardChargesServiceStub{byID: map[string]charges.Charge{
			"flat-fee-charge":    charges.NewCharge(chargesflatfee.Charge{}),
			"usage-based-charge": charges.NewCharge(chargesusagebased.Charge{}),
		}},
		billingService: &chargeReplacementGuardBillingServiceStub{},
	}
}

func TestChargeReplacementGuard(t *testing.T) {
	t.Run("flat fee prior realization on immutable invoice blocks replacement", func(t *testing.T) {
		// given: the current run has no invoice but a prior run belongs to an issued invoice.
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.Subscription.ItemID = "previous-item"
		existingIntent.AmountBeforeProration = existingIntent.AmountBeforeProration.Add(existingIntent.AmountBeforeProration)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		invoiceID := "issued-invoice"
		chargeService := &chargeReplacementGuardChargesServiceStub{byID: map[string]charges.Charge{
			"flat-fee-charge": charges.NewCharge(chargesflatfee.Charge{Realizations: chargesflatfee.Realizations{
				CurrentRun: &chargesflatfee.RealizationRun{},
				PriorRuns: chargesflatfee.RealizationRuns{
					{RealizationRunBase: chargesflatfee.RealizationRunBase{InvoiceID: &invoiceID}},
					{RealizationRunBase: chargesflatfee.RealizationRunBase{InvoiceID: &invoiceID}},
				},
			}}),
		}}
		billingService := &chargeReplacementGuardBillingServiceStub{byID: map[string]billing.StandardInvoice{
			invoiceID: {StandardInvoiceBase: billing.StandardInvoiceBase{Status: billing.StandardInvoiceStatusIssued, StatusDetails: billing.StandardInvoiceStatusDetails{Immutable: true}}},
		}}
		reconciler := &Service{chargesService: chargeService, billingService: billingService}
		patches := newFlatFeeChargeCollection(1)
		references := make(ChargeReferencePatches)

		// when: the changed physical item would replace the old charge.
		err := reconciler.diffItem(t.Context(), &target, existing, patches, references)

		// then: planning rejects replacement before adding either patch.
		require.ErrorContains(t, err, "immutable")
		require.True(t, patches.Patches().IsEmpty())
		require.True(t, references.IsEmpty())
		require.Len(t, chargeService.lookups, 1)
		require.Equal(t, chargesmeta.Expands{chargesmeta.ExpandRealizations, chargesmeta.ExpandDeletedRealizations}, chargeService.lookups[0].Expands)
		require.Len(t, billingService.lookups, 1)
		require.Equal(t, []string{invoiceID}, billingService.lookups[0].IDs)
		require.True(t, billingService.lookups[0].IncludeDeleted)
	})

	t.Run("usage historical realization on immutable invoice blocks replacement", func(t *testing.T) {
		// given: a previous usage run belongs to a paid invoice.
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestUsageRateCard())
		existingIntent := newUsageBasedComparisonTestIntentFromTarget(t, target)
		existingIntent.ServicePeriod.From = existingIntent.ServicePeriod.From.AddDate(0, -1, 0)
		existing := newUsageBasedComparisonTestItem(t, target, "usage-based-charge", existingIntent)
		invoiceID := "paid-invoice"
		chargeService := &chargeReplacementGuardChargesServiceStub{byID: map[string]charges.Charge{
			"usage-based-charge": charges.NewCharge(chargesusagebased.Charge{Realizations: chargesusagebased.RealizationRuns{
				{RealizationRunBase: chargesusagebased.RealizationRunBase{}},
				{RealizationRunBase: chargesusagebased.RealizationRunBase{InvoiceID: &invoiceID}},
			}}),
		}}
		billingService := &chargeReplacementGuardBillingServiceStub{byID: map[string]billing.StandardInvoice{
			invoiceID: {StandardInvoiceBase: billing.StandardInvoiceBase{Status: billing.StandardInvoiceStatusPaid, StatusDetails: billing.StandardInvoiceStatusDetails{Immutable: true}}},
		}}
		reconciler := &Service{chargesService: chargeService, billingService: billingService}
		patches := newUsageBasedChargeCollection(1)

		// when: the service-period start requires replacement.
		err := reconciler.diffItem(t.Context(), &target, existing, patches, make(ChargeReferencePatches))

		// then: the prior paid realization invoice blocks replacement.
		require.ErrorContains(t, err, "immutable")
		require.True(t, patches.Patches().IsEmpty())
		require.Len(t, billingService.lookups, 1)
	})

	t.Run("mutable invoice permits replacement", func(t *testing.T) {
		// given: the old flat fee has only a draft invoice.
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
		existingIntent.ServicePeriod.From = existingIntent.ServicePeriod.From.AddDate(0, -1, 0)
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
		invoiceID := "draft-invoice"
		chargeService := &chargeReplacementGuardChargesServiceStub{byID: map[string]charges.Charge{
			"flat-fee-charge": charges.NewCharge(chargesflatfee.Charge{Realizations: chargesflatfee.Realizations{
				CurrentRun: &chargesflatfee.RealizationRun{RealizationRunBase: chargesflatfee.RealizationRunBase{InvoiceID: &invoiceID}},
			}}),
		}}
		billingService := &chargeReplacementGuardBillingServiceStub{byID: map[string]billing.StandardInvoice{
			invoiceID: {StandardInvoiceBase: billing.StandardInvoiceBase{
				Status: billing.StandardInvoiceStatusDraftCreated,
				StatusDetails: billing.StandardInvoiceStatusDetails{AvailableActions: billing.StandardInvoiceAvailableActions{
					Advance: &billing.StandardInvoiceAvailableActionDetails{ResultingState: billing.StandardInvoiceStatusDraftCollecting},
				}},
			}},
		}}
		reconciler := &Service{chargesService: chargeService, billingService: billingService}
		patches := newFlatFeeChargeCollection(1)

		// when: the service-period start requires replacement.
		err := reconciler.diffItem(t.Context(), &target, existing, patches, make(ChargeReferencePatches))

		// then: the mutable invoice permits the delete and create pair.
		require.NoError(t, err)
		require.Len(t, patches.Patches().PatchesByChargeID, 1)
		require.Len(t, patches.Patches().Creates, 1)
	})

	t.Run("cancellation delete skips the charge replacement guard", func(t *testing.T) {
		// given: a subscription-managed flat fee with no target item.
		target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
		existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", newFlatFeeComparisonTestIntentFromTarget(t, target))
		chargeService := &chargeReplacementGuardChargesServiceStub{err: errors.New("unexpected realization lookup")}
		reconciler := &Service{chargesService: chargeService}
		patches := newFlatFeeChargeCollection(1)

		// when: cancellation removes the target.
		err := reconciler.diffItem(t.Context(), nil, existing, patches, make(ChargeReferencePatches))

		// then: deletion is planned without loading realizations.
		require.NoError(t, err)
		require.Empty(t, chargeService.lookups)
		require.Len(t, patches.Patches().PatchesByChargeID, 1)
		require.Empty(t, patches.Patches().Creates)
	})
}

func TestChargeReplacementGuardLookupFailsClosed(t *testing.T) {
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
	existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
	existingIntent.ServicePeriod.From = existingIntent.ServicePeriod.From.AddDate(0, -1, 0)
	existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
	invoiceID := "historical-invoice"

	tests := []struct {
		name        string
		chargeError error
		listError   error
		invoices    map[string]billing.StandardInvoice
		extra       []billing.StandardInvoice
		wantError   string
	}{
		{name: "charge realization lookup", chargeError: errors.New("charge unavailable"), wantError: "charge unavailable"},
		{name: "invoice batch lookup", listError: errors.New("invoices unavailable"), wantError: "invoices unavailable"},
		{name: "missing invoice", wantError: "only 0 of 1 realization invoices were returned"},
		{
			name: "unexpected invoice",
			invoices: map[string]billing.StandardInvoice{
				invoiceID: {StandardInvoiceBase: billing.StandardInvoiceBase{
					Status:        billing.StandardInvoiceStatusDraftInvalid,
					StatusDetails: billing.StandardInvoiceStatusDetails{Failed: true},
				}},
			},
			extra:     []billing.StandardInvoice{{StandardInvoiceBase: billing.StandardInvoiceBase{ID: "other-invoice"}}},
			wantError: "unexpected realization invoice[other-invoice]",
		},
		{
			name: "unresolved status details",
			invoices: map[string]billing.StandardInvoice{
				invoiceID: {StandardInvoiceBase: billing.StandardInvoiceBase{Status: billing.StandardInvoiceStatusIssued}},
			},
			wantError: "has no resolved status details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: replacing the charge requires a historical invoice lookup.
			chargeService := &chargeReplacementGuardChargesServiceStub{
				byID: map[string]charges.Charge{
					"flat-fee-charge": charges.NewCharge(chargesflatfee.Charge{Realizations: chargesflatfee.Realizations{
						PriorRuns: chargesflatfee.RealizationRuns{{RealizationRunBase: chargesflatfee.RealizationRunBase{InvoiceID: &invoiceID}}},
					}}),
				},
				err: tt.chargeError,
			}
			billingService := &chargeReplacementGuardBillingServiceStub{byID: tt.invoices, extra: tt.extra, err: tt.listError}
			reconciler := &Service{chargesService: chargeService, billingService: billingService}
			patches := newFlatFeeChargeCollection(1)

			// when: planning checks whether replacement is safe.
			err := reconciler.diffItem(t.Context(), &target, existing, patches, make(ChargeReferencePatches))

			// then: unavailable or incomplete realization invoices block replacement.
			require.ErrorContains(t, err, tt.wantError)
			require.True(t, patches.Patches().IsEmpty())
		})
	}
}

func TestChargeReplacementGuardSkipsReferenceRepairAndShrink(t *testing.T) {
	target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, newChargePatchTestFlatRateCard())
	existingIntent := newFlatFeeComparisonTestIntentFromTarget(t, target)
	existingIntent.Subscription.ItemID = "previous-item"
	existingIntent.ServicePeriod.To = existingIntent.ServicePeriod.To.AddDate(0, 1, 0)
	existingIntent.FullServicePeriod.To = existingIntent.FullServicePeriod.To.AddDate(0, 1, 0)
	existingIntent.BillingPeriod.To = existingIntent.BillingPeriod.To.AddDate(0, 1, 0)
	existing := newFlatFeeComparisonTestItem(t, target, "flat-fee-charge", existingIntent)
	chargeService := &chargeReplacementGuardChargesServiceStub{err: errors.New("unexpected realization lookup")}
	reconciler := &Service{chargesService: chargeService}
	patches := newFlatFeeChargeCollection(1)
	references := make(ChargeReferencePatches)

	// given: the physical item changed, while billing terms and the service-period start match.
	// when: reconciliation repairs the reference and shortens the existing charge.
	err := reconciler.diffItem(t.Context(), &target, existing, patches, references)

	// then: the charge replacement guard does not load realizations.
	require.NoError(t, err)
	require.Empty(t, chargeService.lookups)
	require.Len(t, references, 1)
	require.Len(t, patches.Patches().PatchesByChargeID, 1)
	require.Empty(t, patches.Patches().Creates)
}
