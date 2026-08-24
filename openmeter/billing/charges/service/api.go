package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

func (s *service) CreateCustomerCharge(ctx context.Context, input charges.CreateCustomerChargeInput) (charges.CustomerCharge, error) {
	if err := input.Validate(); err != nil {
		return charges.CustomerCharge{}, err
	}

	currency, err := s.currencyResolver.ResolveCurrency(ctx, input.Namespace, currencies.CurrencyRef{
		Code: input.CurrencyCode,
	})
	if err != nil {
		return charges.CustomerCharge{}, fmt.Errorf("resolving currency: %w", err)
	}

	if currency.IsCustom() {
		return charges.CustomerCharge{}, models.NewGenericValidationError(fmt.Errorf("currency: %w", meta.ErrCustomCurrencyNotSupported))
	}

	intent := meta.Intent{
		ManagedBy:         billing.ManuallyManagedLine,
		CustomerID:        input.CustomerID,
		Currency:          *currency,
		TaxConfig:         input.TaxConfig,
		UniqueReferenceID: input.UniqueReferenceID,
	}

	var chargeIntent charges.ChargeIntent
	switch {
	case input.FlatFee != nil:
		chargeIntent = charges.NewChargeIntent(flatfee.Intent{
			Intent:              intent,
			IntentMutableFields: input.FlatFee.IntentMutableFields,
			FeatureID:           input.FlatFee.FeatureID,
			SettlementMode:      input.FlatFee.SettlementMode,
		})
	case input.UsageBased != nil:
		chargeIntent = charges.NewChargeIntent(usagebased.Intent{
			Intent:              intent,
			IntentMutableFields: input.UsageBased.IntentMutableFields,
			FeatureID:           input.UsageBased.FeatureID,
			SettlementMode:      input.UsageBased.SettlementMode,
		})
	}

	created, err := s.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents:   charges.ChargeIntents{chargeIntent},
	})
	if err != nil {
		return charges.CustomerCharge{}, err
	}

	if len(created) != 1 {
		return charges.CustomerCharge{}, fmt.Errorf("expected one created charge, got %d", len(created))
	}

	// The API response includes the realization view (the whole service period
	// is outstanding on a fresh charge), so it is built here just like on the
	// list path. No expands apply on the create path.
	return buildCustomerCharge(created[0], customerChargeEntities{})
}

func (s *service) DeleteCustomerCharge(ctx context.Context, input charges.DeleteCustomerChargeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return err
	}

	policy, err := resolveDeletePolicy(input.PaymentAdjustment)
	if err != nil {
		return err
	}

	patch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Policy:       policy,
	})
	if err != nil {
		return fmt.Errorf("creating charge delete patch: %w", err)
	}

	return s.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
		PatchesByChargeID: map[string]charges.Patch{
			input.ChargeID: patch,
		},
	})
}

func resolveDeletePolicy(adjustment charges.PaymentAdjustment) (meta.PatchDeletePolicy, error) {
	switch adjustment {
	case charges.PaymentAdjustmentNone:
		return meta.PatchDeletePolicy{
			CreditRefundPolicy:  meta.CreditRefundPolicyIgnore,
			InvoiceRefundPolicy: meta.InvoiceRefundPolicyIgnore,
		}, nil
	default:
		return meta.PatchDeletePolicy{}, fmt.Errorf("unsupported payment adjustment: %s", adjustment)
	}
}

func (s *service) SetCustomerChargeOverride(ctx context.Context, input charges.SetCustomerChargeOverrideInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return charges.Charge{}, err
	}

	chargeID := meta.ChargeID{
		Namespace: input.Namespace,
		ID:        input.ChargeID,
	}

	existing, err := s.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
	if err != nil {
		return charges.Charge{}, err
	}

	customerID, err := existing.GetCustomerID()
	if err != nil {
		return charges.Charge{}, fmt.Errorf("getting charge customer: %w", err)
	}

	if customerID.ID != input.CustomerID {
		return charges.Charge{}, models.NewGenericNotFoundError(errors.New("charge not found"))
	}

	var patch charges.Patch
	switch existing.Type() {
	case meta.ChargeTypeFlatFee:
		if input.FlatFee == nil {
			return charges.Charge{}, models.NewGenericValidationError(fmt.Errorf("flat fee override fields are required for flat fee charge %s", input.ChargeID))
		}

		patch, err = meta.NewPatchSetOverride(flatfee.NewPatchSetOverrideInput{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: *input.FlatFee,
		})
	case meta.ChargeTypeUsageBased:
		if input.UsageBased == nil {
			return charges.Charge{}, models.NewGenericValidationError(fmt.Errorf("usage based override fields are required for usage based charge %s", input.ChargeID))
		}

		patch, err = meta.NewPatchSetOverride(usagebased.NewPatchSetOverrideInput{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: *input.UsageBased,
		})
	case meta.ChargeTypeCreditPurchase:
		return charges.Charge{}, models.NewGenericValidationError(errors.New("setting overrides for credit purchase charges is not supported"))
	default:
		return charges.Charge{}, fmt.Errorf("unsupported charge type: %s", existing.Type())
	}
	if err != nil {
		return charges.Charge{}, fmt.Errorf("creating charge override patch: %w", err)
	}

	if err := s.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			input.ChargeID: patch,
		},
	}); err != nil {
		return charges.Charge{}, err
	}

	return s.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
}

func (s *service) ClearCustomerChargeOverride(ctx context.Context, input charges.ClearCustomerChargeOverrideInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return charges.Charge{}, err
	}

	chargeID := meta.ChargeID{
		Namespace: input.Namespace,
		ID:        input.ChargeID,
	}

	existing, err := s.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
	if err != nil {
		return charges.Charge{}, err
	}

	customerID, err := existing.GetCustomerID()
	if err != nil {
		return charges.Charge{}, fmt.Errorf("getting charge customer: %w", err)
	}

	if customerID.ID != input.CustomerID {
		return charges.Charge{}, models.NewGenericNotFoundError(errors.New("charge not found"))
	}

	switch existing.Type() {
	case meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased:
	case meta.ChargeTypeCreditPurchase:
		return charges.Charge{}, models.NewGenericValidationError(errors.New("clearing overrides for credit purchase charges is not supported"))
	default:
		return charges.Charge{}, fmt.Errorf("unsupported charge type: %s", existing.Type())
	}

	patch, err := meta.NewPatchClearOverride(meta.NewPatchClearOverrideInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
	})
	if err != nil {
		return charges.Charge{}, fmt.Errorf("creating charge clear override patch: %w", err)
	}

	if err := s.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			input.ChargeID: patch,
		},
	}); err != nil {
		return charges.Charge{}, err
	}

	return s.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
}

func (s *service) ListCustomerCharges(ctx context.Context, input charges.ListCustomerChargesInput) (charges.ListCustomerChargesResult, error) {
	if err := input.Validate(); err != nil {
		return charges.ListCustomerChargesResult{}, err
	}

	listInput := input.ListChargesInput
	// Realization runs always load: booked totals and the resolved realization
	// view depend on them. Deleted runs load too, so voided history surfaces
	// as audit entries.
	listInput.Expands = listInput.Expands.
		With(meta.ExpandRealizations).
		With(meta.ExpandDeletedRealizations)

	listed, err := s.ListCharges(ctx, listInput)
	if err != nil {
		return charges.ListCustomerChargesResult{}, err
	}

	refs, err := collectCustomerChargeReferences(listed.Items)
	if err != nil {
		return charges.ListCustomerChargesResult{}, err
	}

	entities, err := s.loadCustomerChargeEntities(ctx, input.Namespace, input.CustomerIDs[0], refs, listInput.Expands)
	if err != nil {
		return charges.ListCustomerChargesResult{}, err
	}

	customerCharges, err := lo.MapErr(listed.Items, func(charge charges.Charge, _ int) (charges.CustomerCharge, error) {
		return buildCustomerCharge(charge, entities)
	})
	if err != nil {
		return charges.ListCustomerChargesResult{}, err
	}

	return charges.ListCustomerChargesResult{
		Charges: pagination.Result[charges.CustomerCharge]{
			Page:       listed.Page,
			TotalCount: listed.TotalCount,
			Items:      customerCharges,
		},
		Expands: listInput.Expands,
	}, nil
}

// buildCustomerCharge assembles the API-facing CustomerCharge from the
// domain charge and the entities loaded for the applied expands: the
// side-loaded Customer/Feature/Subscription (nil unless their expand was
// applied) and the charge-type-matching resolved realization history (with
// per-run invoices when the realization-invoice expand loaded them). Credit
// purchase charges only receive the customer.
func buildCustomerCharge(charge charges.Charge, entities customerChargeEntities) (charges.CustomerCharge, error) {
	out := charges.CustomerCharge{
		Charge: charge,
		// The listing is scoped to a single customer, so every charge on the
		// page shares the one loaded customer (nil without the expand).
		Customer: entities.customer,
	}

	switch charge.Type() {
	case meta.ChargeTypeUsageBased:
		ub, err := charge.AsUsageBasedCharge()
		if err != nil {
			return charges.CustomerCharge{}, err
		}

		resolved, err := resolveUsageBasedRealizations(ub, entities.invoiceLinesByID)
		if err != nil {
			return charges.CustomerCharge{}, fmt.Errorf("charge %s: resolving realizations: %w", ub.ID, err)
		}

		out.UsageBasedRealizations = resolved

		if feat, ok := entities.featuresByRef[ub.GetFeatureKeyOrID()]; ok {
			out.Feature = &feat
		}

		if sub := ub.Intent.GetSubscription(); sub != nil {
			if subEntity, ok := entities.subscriptionsByID[sub.SubscriptionID]; ok {
				out.Subscription = &subEntity
			}
		}
	case meta.ChargeTypeFlatFee:
		ff, err := charge.AsFlatFeeCharge()
		if err != nil {
			return charges.CustomerCharge{}, err
		}

		resolved, err := resolveFlatFeeRealizations(ff, entities.invoiceLinesByID)
		if err != nil {
			return charges.CustomerCharge{}, fmt.Errorf("charge %s: resolving realizations: %w", ff.ID, err)
		}

		out.FlatFeeRealizations = resolved

		if featureRef := ff.GetFeatureRef(); featureRef != nil {
			if feat, ok := entities.featuresByRef[*featureRef]; ok {
				out.Feature = &feat
			}
		}

		if sub := ff.Intent.GetSubscription(); sub != nil {
			if subEntity, ok := entities.subscriptionsByID[sub.SubscriptionID]; ok {
				out.Subscription = &subEntity
			}
		}
	}

	return out, nil
}

// customerChargeReferences collects the entity references a page of charges
// points at, so the facade bulk-loads each kind once.
type customerChargeReferences struct {
	featureRefs     []ref.IDOrKey
	subscriptionIDs []string
	invoiceIDs      []string
}

func collectCustomerChargeReferences(items charges.Charges) (customerChargeReferences, error) {
	out := customerChargeReferences{}

	for _, item := range items {
		switch item.Type() {
		case meta.ChargeTypeUsageBased:
			ub, err := item.AsUsageBasedCharge()
			if err != nil {
				return customerChargeReferences{}, err
			}

			if featureRef := ub.GetFeatureKeyOrID(); featureRef != (ref.IDOrKey{}) {
				out.featureRefs = append(out.featureRefs, featureRef)
			}

			if sub := ub.Intent.GetSubscription(); sub != nil {
				out.subscriptionIDs = append(out.subscriptionIDs, sub.SubscriptionID)
			}

			for _, run := range ub.Realizations {
				if run.InvoiceID != nil {
					out.invoiceIDs = append(out.invoiceIDs, *run.InvoiceID)
				}
			}
		case meta.ChargeTypeFlatFee:
			ff, err := item.AsFlatFeeCharge()
			if err != nil {
				return customerChargeReferences{}, err
			}

			if featureRef := ff.GetFeatureRef(); featureRef != nil {
				out.featureRefs = append(out.featureRefs, *featureRef)
			}

			if sub := ff.Intent.GetSubscription(); sub != nil {
				out.subscriptionIDs = append(out.subscriptionIDs, sub.SubscriptionID)
			}

			runs := ff.Realizations.PriorRuns
			if ff.Realizations.CurrentRun != nil {
				runs = append(slices.Clone(runs), *ff.Realizations.CurrentRun)
			}
			for _, run := range runs {
				if run.InvoiceID != nil {
					out.invoiceIDs = append(out.invoiceIDs, *run.InvoiceID)
				}
			}
		}
	}

	out.featureRefs = lo.Uniq(out.featureRefs)
	out.subscriptionIDs = lo.Uniq(out.subscriptionIDs)
	out.invoiceIDs = lo.Uniq(out.invoiceIDs)

	return out, nil
}

// customerChargeEntities holds the entities loaded per applied expand; the
// customer stays nil and the maps stay empty when their expand was not
// applied. The listing is scoped to exactly one customer, so it is a single
// entity rather than a map.
type customerChargeEntities struct {
	customer          *customer.Customer
	featuresByRef     map[ref.IDOrKey]feature.Feature
	subscriptionsByID map[string]subscription.Subscription
	// invoiceLinesByID pairs each realized invoice line with its parent
	// invoice header, keyed by line ID: realization runs book to a specific
	// line, and two runs may share an invoice through different lines, so the
	// invoice ID alone cannot key the pairing.
	invoiceLinesByID map[string]billing.StandardLineWithInvoiceHeader
}

func (s *service) loadCustomerChargeEntities(ctx context.Context, namespace string, customerID string, refs customerChargeReferences, expands meta.Expands) (customerChargeEntities, error) {
	entities := customerChargeEntities{}
	var err error

	if expands.Has(meta.ExpandCustomer) {
		entities.customer, err = s.getCustomerChargeCustomer(ctx, namespace, customerID)
		if err != nil {
			return customerChargeEntities{}, fmt.Errorf("loading customer: %w", err)
		}
	}

	if expands.Has(meta.ExpandFeature) {
		entities.featuresByRef, err = s.listCustomerChargeFeatures(ctx, namespace, refs.featureRefs)
		if err != nil {
			return customerChargeEntities{}, fmt.Errorf("loading features: %w", err)
		}
	}

	if expands.Has(meta.ExpandSubscription) {
		entities.subscriptionsByID, err = s.listCustomerChargeSubscriptions(ctx, namespace, customerID, refs.subscriptionIDs)
		if err != nil {
			return customerChargeEntities{}, fmt.Errorf("loading subscriptions: %w", err)
		}
	}

	if expands.Has(meta.ExpandRealizationInvoice) {
		entities.invoiceLinesByID, err = s.listRealizationInvoiceLines(ctx, namespace, customerID, refs.invoiceIDs)
		if err != nil {
			return customerChargeEntities{}, fmt.Errorf("loading realization invoices: %w", err)
		}
	}

	return entities, nil
}

// getCustomerChargeCustomer loads the listing's single customer for the
// customer expand. It goes through ListCustomers instead of GetCustomer
// because charges outlive their customer and deleted customers must still
// expand; a missing customer resolves to nil so the wire falls back to the
// id reference instead of failing the listing.
func (s *service) getCustomerChargeCustomer(ctx context.Context, namespace string, id string) (*customer.Customer, error) {
	listed, err := s.customerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:      namespace,
		Page:           pagination.NewPage(1, 1),
		IncludeDeleted: true,
		CustomerIDs:    []string{id},
	})
	if err != nil {
		return nil, fmt.Errorf("listing customers: %w", err)
	}

	if len(listed.Items) == 0 {
		return nil, nil
	}

	return lo.ToPtr(listed.Items[0]), nil
}

// listCustomerChargeFeatures bulk-loads the referenced features for the
// feature expand. Refs mix resolved IDs (active charges) and keys (created
// charges), which ResolveFeatureMeters handles natively.
func (s *service) listCustomerChargeFeatures(ctx context.Context, namespace string, refs []ref.IDOrKey) (map[ref.IDOrKey]feature.Feature, error) {
	out := make(map[ref.IDOrKey]feature.Feature, len(refs))
	if len(refs) == 0 {
		return out, nil
	}

	meterRefs := lo.Map(refs, func(featureRef ref.IDOrKey, _ int) feature.FeatureMeterRef {
		return feature.FeatureMeterRef{IDOrKey: featureRef}
	})

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, namespace, meterRefs...)
	if err != nil {
		return nil, fmt.Errorf("resolving features: %w", err)
	}

	for _, featureRef := range refs {
		featureMeter, err := featureMeters.Resolve(feature.FeatureMeterRef{IDOrKey: featureRef})
		if err != nil {
			// A stale reference must not fail the listing: the converter
			// falls back to the id reference, mirroring how deleted
			// customers and missing subscriptions degrade.
			if models.IsGenericNotFoundError(err) {
				continue
			}

			return nil, fmt.Errorf("resolving feature %v: %w", featureRef, err)
		}

		out[featureRef] = featureMeter.Feature
	}

	return out, nil
}

// listCustomerChargeSubscriptions bulk-loads the referenced subscriptions for
// the subscription expand. The customer filter is defense-in-depth: the IDs
// come from system-written charge references already scoped to the customer,
// but an upstream integrity bug must not let another customer's subscription
// ride along in this customer's listing.
func (s *service) listCustomerChargeSubscriptions(ctx context.Context, namespace string, customerID string, ids []string) (map[string]subscription.Subscription, error) {
	out := make(map[string]subscription.Subscription, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	listed, err := s.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces:     []string{namespace},
		Page:           pagination.NewPage(1, len(ids)),
		IncludeDeleted: true,
		ID:             &filter.FilterULID{FilterString: filter.FilterString{In: &ids}},
		CustomerID:     &filter.FilterULID{FilterString: filter.FilterString{Eq: &customerID}},
	})
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}

	for _, item := range listed.Items {
		out[item.ID] = item
	}

	return out, nil
}

// listRealizationInvoiceLines loads the invoices referenced by realization
// runs and breaks them into per-line entries keyed by line ID, each pairing
// the line with its parent invoice header (lines stripped): realization runs
// book to a specific line, so the line is the entity the expand resolves.
// There are no ID-only stubs without the expand; converters fall back to the
// run's invoice ID. The customer filter is defense-in-depth against another
// customer's invoice riding along through a corrupted run reference.
func (s *service) listRealizationInvoiceLines(ctx context.Context, namespace string, customerID string, ids []string) (map[string]billing.StandardLineWithInvoiceHeader, error) {
	out := make(map[string]billing.StandardLineWithInvoiceHeader)
	if len(ids) == 0 {
		return out, nil
	}

	listed, err := s.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespace:      namespace,
		Page:           pagination.NewPage(1, len(ids)),
		IncludeDeleted: true,
		IDs:            ids,
		CustomerID:     &filter.FilterULID{FilterString: filter.FilterString{Eq: &customerID}},
		Expand:         billing.InvoiceExpandAll,
	})
	if err != nil {
		return nil, fmt.Errorf("listing invoices: %w", err)
	}

	for _, item := range listed.Items {
		// Realization runs only ever book to standard invoices; anything else
		// carries no bookable lines, so it cannot satisfy a run reference.
		if item.Type() != billing.InvoiceTypeStandard {
			continue
		}

		std, err := item.AsStandardInvoice()
		if err != nil {
			return nil, fmt.Errorf("reading invoice: %w", err)
		}

		header := std
		header.Lines = billing.StandardInvoiceLines{}

		for _, line := range std.Lines.OrEmpty() {
			if line == nil {
				continue
			}

			out[line.ID] = billing.StandardLineWithInvoiceHeader{
				Line:    line,
				Invoice: header,
			}
		}
	}

	return out, nil
}
