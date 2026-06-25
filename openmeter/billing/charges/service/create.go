package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type chargesWithInvoiceNowActions struct {
	charges                          charges.Charges
	collectionAlignmentBypassedLines []invoicePendingLinesInput
	pendingLineResults               []*billing.CreatePendingInvoiceLinesResult
}

// applyDefaultTaxCodes fills in nil TaxCodeID on each intent's TaxConfig using the namespace's
// organization default tax codes. Invoicing default applies to flat-fee and usage-based charges;
// credit-grant default applies to credit purchase charges. Fails if any intent needs the fallback
// but the namespace has no defaults provisioned.
func (s *service) applyDefaultTaxCodes(ctx context.Context, namespace string, intents charges.ChargeIntents) (charges.ChargeIntents, error) {
	getDefaultTaxCodes := sync.OnceValues(func() (taxcode.OrganizationDefaultTaxCodes, error) {
		return s.taxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: namespace,
		})
	})

	return slicesx.MapWithErr(intents, func(intent charges.ChargeIntent) (charges.ChargeIntent, error) {
		var taxCodeID string

		switch intent.Type() {
		case meta.ChargeTypeFlatFee:
			flatFee, err := intent.AsFlatFeeIntent()
			if err != nil {
				return charges.ChargeIntent{}, err
			}
			taxCodeID = flatFee.TaxConfig.TaxCodeID
		case meta.ChargeTypeCreditPurchase:
			creditPurchase, err := intent.AsCreditPurchaseIntent()
			if err != nil {
				return charges.ChargeIntent{}, err
			}
			taxCodeID = creditPurchase.TaxConfig.TaxCodeID
		case meta.ChargeTypeUsageBased:
			usageBased, err := intent.AsUsageBasedIntent()
			if err != nil {
				return charges.ChargeIntent{}, err
			}
			taxCodeID = usageBased.TaxConfig.TaxCodeID
		default:
			return charges.ChargeIntent{}, fmt.Errorf("unsupported charge type: %s", intent.Type())
		}

		if taxCodeID != "" {
			return intent, nil
		}

		defaultTaxCodes, err := getDefaultTaxCodes()
		if err != nil {
			return charges.ChargeIntent{}, err
		}

		// credit purchases use the credit-grant default; flat-fee and usage-based use the invoicing default
		defaultID := defaultTaxCodes.InvoicingTaxCodeID
		if intent.Type() == meta.ChargeTypeCreditPurchase {
			defaultID = defaultTaxCodes.CreditGrantTaxCodeID
		}

		return intent.WithTaxCodeID(defaultID)
	})
}

func (s *service) Create(ctx context.Context, input charges.CreateInput) (charges.Charges, error) {
	result, err := s.create(ctx, input)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	// TODO: once we have proper state machine for credit purchases, we can remove this and make the
	// autoAdvanceCreatedCharges handle the invoice now actions.
	if len(result.collectionAlignmentBypassedLines) > 0 {
		if err := s.invokeInvoiceNowOnCreate(ctx, result.collectionAlignmentBypassedLines); err != nil {
			return nil, fmt.Errorf("invoking invoice now on create: %w", err)
		}
	}

	return s.autoAdvanceCreatedCharges(ctx, result.charges)
}

func (s *service) create(ctx context.Context, input charges.CreateInput) (*chargesWithInvoiceNowActions, error) {
	if input.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return nil, err
	}

	intentsWithDefaults, err := s.applyDefaultTaxCodes(ctx, input.Namespace, input.Intents)
	if err != nil {
		return nil, err
	}
	input.Intents = intentsWithDefaults

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (*chargesWithInvoiceNowActions, error) {
		intentsByType, err := input.Intents.ByType()
		if err != nil {
			return nil, err
		}

		featureKeys, err := input.Intents.CollectFeatureKeys()
		if err != nil {
			return nil, err
		}

		createFeatureMeters, err := s.featureService.ResolveFeatureMeters(ctx, input.Namespace, lo.Map(featureKeys, func(featureKey string, _ int) ref.IDOrKey {
			return ref.IDOrKey{Key: featureKey}
		})...)
		if err != nil {
			return nil, fmt.Errorf("resolve create feature meters: %w", err)
		}

		createdCharges := make([]charges.WithIndex[charges.Charge], 0, len(input.Intents))
		gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(input.Intents))

		// Let's create all the flat fee charges in bulk and record any gathering lines to create
		flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.FlatFee, func(intent charges.WithIndex[flatfee.Intent], _ int) flatfee.Intent {
				return intent.Value
			}),
			FeatureMeters: createFeatureMeters,
		})
		if err != nil {
			return nil, err
		}

		createdCharges = append(
			createdCharges,
			lo.Map(flatFees, func(fee flatfee.ChargeWithGatheringLine, idx int) charges.WithIndex[charges.Charge] {
				return charges.WithIndex[charges.Charge]{
					Index: intentsByType.FlatFee[idx].Index,
					Value: charges.NewCharge(fee.Charge),
				}
			})...,
		)

		for _, fee := range flatFees {
			if fee.GatheringLineToCreate != nil {
				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *fee.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        fee.Charge.Intent.CustomerID,
					},
				})
			}
		}

		// Let's create all the usage based charges in bulk
		usageBasedCharges, err := s.usageBasedService.Create(ctx, usagebased.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.UsageBased, func(intent charges.WithIndex[usagebased.Intent], _ int) usagebased.Intent {
				return intent.Value
			}),
			FeatureMeters: createFeatureMeters,
		})
		if err != nil {
			return nil, err
		}

		createdCharges = append(
			createdCharges,
			lo.Map(usageBasedCharges, func(charge usagebased.ChargeWithGatheringLine, idx int) charges.WithIndex[charges.Charge] {
				return charges.WithIndex[charges.Charge]{
					Index: intentsByType.UsageBased[idx].Index,
					Value: charges.NewCharge(charge.Charge),
				}
			})...,
		)

		for _, charge := range usageBasedCharges {
			if charge.GatheringLineToCreate != nil {
				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *charge.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        charge.Charge.Intent.CustomerID,
					},
				})
			}
		}

		// Let's create all the credit purchase charges
		for _, intent := range intentsByType.CreditPurchase {
			result, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: input.Namespace,
				Intent:    intent.Value,
			})
			if err != nil {
				return nil, err
			}

			// For invoice settlement, prepare the gathering line (actual invoicing happens after TX commits)
			if result.GatheringLineToCreate != nil {
				bypassCollectionAlignment := false
				if !result.Charge.Intent.ServicePeriod.From.After(clock.Now()) {
					// Credit purchases are not standard invoices, so for now we bypass the collection
					// period here to make sure they are billed immediately once effective. Gathering
					// line collection will not behave the same way. If we get customer feedback, we can
					// later fine-tune this behavior.
					bypassCollectionAlignment = true
				}

				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *result.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        result.Charge.Intent.CustomerID,
					},
					BypassCollectionAlignment: bypassCollectionAlignment,
				})
			}

			createdCharges = append(createdCharges, charges.WithIndex[charges.Charge]{
				Index: intent.Index,
				Value: charges.NewCharge(result.Charge),
			})
		}

		// Let's generate the gathering lines for the flat fees
		gatheringLineResult, err := s.createGatheringLines(ctx, gatheringLinesToCreate)
		if err != nil {
			return nil, err
		}

		// Let's map the created charges to the original intents
		result := make([]charges.Charge, len(input.Intents))
		for _, createdCharge := range createdCharges {
			result[createdCharge.Index] = createdCharge.Value
		}

		if err := s.recognizeCreatedCreditPurchaseEarnings(ctx, result); err != nil {
			return nil, err
		}

		return &chargesWithInvoiceNowActions{
			charges:                          result,
			collectionAlignmentBypassedLines: gatheringLineResult.collectionAlignmentBypassedLines,
			pendingLineResults:               gatheringLineResult.pendingLineResults,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// autoAdvanceCreatedCharges post-processes newly created charges
// it handles credit-only usage-based and flat fee charges whose next advancement is already due.
// a separate transaction is used to make sure that we persist the creation state even if the advancing fails (as
// a worker will try to advance the charges again).
func (s *service) autoAdvanceCreatedCharges(ctx context.Context, created charges.Charges) (charges.Charges, error) {
	// Collect unique customer IDs that have newly created credit-only charges.
	customerIDs := make(map[customer.CustomerID]struct{})
	for _, c := range created {
		switch c.Type() {
		case meta.ChargeTypeUsageBased:
			ub, err := c.AsUsageBasedCharge()
			if err != nil {
				return nil, err
			}

			if ub.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
				continue
			}

			if !isAdvanceDue(ub.State.AdvanceAfter) {
				continue
			}

			customerIDs[customer.CustomerID{Namespace: ub.Namespace, ID: ub.Intent.CustomerID}] = struct{}{}

		case meta.ChargeTypeFlatFee:
			ff, err := c.AsFlatFeeCharge()
			if err != nil {
				return nil, err
			}

			if ff.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
				continue
			}

			if !isAdvanceDue(ff.State.AdvanceAfter) {
				continue
			}

			customerIDs[customer.CustomerID{Namespace: ff.Namespace, ID: ff.Intent.CustomerID}] = struct{}{}
		}
	}

	if len(customerIDs) == 0 {
		return created, nil
	}

	advancedByID := make(map[string]charges.Charge)
	for custID := range customerIDs {
		advancedCharges, err := s.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: custID,
		})
		if err != nil {
			return nil, fmt.Errorf("auto-advance charges for customer %s: %w", custID.ID, err)
		}

		for _, advanced := range advancedCharges {
			chargeID, err := advanced.GetChargeID()
			if err != nil {
				return nil, err
			}
			advancedByID[chargeID.ID] = advanced
		}
	}

	if len(advancedByID) == 0 {
		return created, nil
	}

	out := make(charges.Charges, len(created))
	for i, c := range created {
		chargeID, err := c.GetChargeID()
		if err != nil {
			return nil, err
		}

		if advanced, ok := advancedByID[chargeID.ID]; ok {
			out[i] = advanced
		} else {
			out[i] = c
		}
	}

	return out, nil
}

func isAdvanceDue(advanceAfter *time.Time) bool {
	return advanceAfter == nil || !clock.Now().Before(*advanceAfter)
}

type currencyAndCustomerID struct {
	currency   currencyx.Code
	customerID customer.CustomerID
}

type gatheringLineWithCustomerID struct {
	gatheringLine             billing.GatheringLine
	customerID                customer.CustomerID
	BypassCollectionAlignment bool
}

func (s *service) invokeInvoiceNowOnCreate(ctx context.Context, invoiceNowLines []invoicePendingLinesInput) error {
	if len(invoiceNowLines) == 0 {
		return nil
	}

	invoiceNowArgs := lo.GroupByMap(invoiceNowLines, func(item invoicePendingLinesInput) (customer.CustomerID, string) {
		return item.CustomerID, item.LineID
	})

	for customerID, lines := range invoiceNowArgs {
		if _, err := s.billingService.InvoicePendingLines(
			ctx,
			billing.InvoicePendingLinesInput{
				Customer:            customerID,
				IncludePendingLines: mo.Some(lines),
				AsOf:                lo.ToPtr(clock.Now()),
			},
			billing.WithBypassCollectionAlignment(),
		); err != nil {
			return fmt.Errorf("invoking invoice now on create: %w", err)
		}
	}

	return nil
}

type invoicePendingLinesInput struct {
	CustomerID customer.CustomerID
	LineID     string
}

type createGatheringLinesResult struct {
	collectionAlignmentBypassedLines []invoicePendingLinesInput
	pendingLineResults               []*billing.CreatePendingInvoiceLinesResult
}

func (s *service) createGatheringLines(ctx context.Context, gatheringLinesToCreate []gatheringLineWithCustomerID) (createGatheringLinesResult, error) {
	if len(gatheringLinesToCreate) == 0 {
		return createGatheringLinesResult{}, nil
	}

	gatheringLinesByCurrencyAndCustomer := lo.GroupBy(gatheringLinesToCreate, func(item gatheringLineWithCustomerID) currencyAndCustomerID {
		return currencyAndCustomerID{
			currency:   item.gatheringLine.Currency,
			customerID: item.customerID,
		}
	})

	out := createGatheringLinesResult{
		collectionAlignmentBypassedLines: make([]invoicePendingLinesInput, 0, len(gatheringLinesToCreate)),
		pendingLineResults:               make([]*billing.CreatePendingInvoiceLinesResult, 0, len(gatheringLinesByCurrencyAndCustomer)),
	}

	for custAndCurrency, lines := range gatheringLinesByCurrencyAndCustomer {
		// Let's create the gathering invoice on invoicing side
		result, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: custAndCurrency.customerID,
			Currency: custAndCurrency.currency,
			Lines: lo.Map(lines, func(item gatheringLineWithCustomerID, _ int) billing.GatheringLine {
				return item.gatheringLine
			}),
		})
		if err != nil {
			return createGatheringLinesResult{}, fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}
		if result == nil {
			return createGatheringLinesResult{}, fmt.Errorf("creating pending invoice lines for charges: result is nil")
		}

		out.pendingLineResults = append(out.pendingLineResults, result)

		// Correlate the returned lines back to their inputs by charge ID rather than by
		// position: billing may drop lines (e.g. zero-amount lines), which would make
		// index-based correlation silently read BypassCollectionAlignment from the wrong line.
		bypassChargeIDs := make(map[string]struct{}, len(lines))
		for _, line := range lines {
			if !line.BypassCollectionAlignment {
				continue
			}

			if line.gatheringLine.ChargeID == nil {
				return createGatheringLinesResult{}, fmt.Errorf("creating pending invoice lines for charges: bypass collection alignment requested for line without charge ID")
			}

			bypassChargeIDs[*line.gatheringLine.ChargeID] = struct{}{}
		}

		for _, line := range result.Lines {
			if line.ChargeID == nil {
				continue
			}

			if _, ok := bypassChargeIDs[*line.ChargeID]; ok {
				out.collectionAlignmentBypassedLines = append(out.collectionAlignmentBypassedLines, invoicePendingLinesInput{
					CustomerID: custAndCurrency.customerID,
					LineID:     line.ID,
				})
			}
		}
	}

	return out, nil
}
