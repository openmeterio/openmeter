package billinginvoices

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingcommon"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/handlers/tax"
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/set"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// ToAPIBillingInvoice converts a billing.Invoice domain union to the v3 API type.
func ToAPIBillingInvoice(inv billing.Invoice) (apiv3models.BillingInvoice, error) {
	switch inv.Type() {
	case billing.InvoiceTypeStandard:
		std, err := inv.AsStandardInvoice()
		if err != nil {
			return apiv3models.BillingInvoice{}, fmt.Errorf("reading standard invoice: %w", err)
		}

		return ToAPIStandardInvoice(std)
	default:
		genericInv, _ := inv.AsGenericInvoice()

		return apiv3models.BillingInvoice{}, billing.NotFoundError{
			ID:     genericInv.GetID(),
			Entity: billing.EntityInvoice,
			Err:    fmt.Errorf("unsupported invoice type %q", inv.Type()),
		}
	}
}

func ToAPIStandardInvoice(std billing.StandardInvoice) (apiv3models.BillingInvoice, error) {
	var out apiv3models.BillingInvoice
	stdAPI, err := toAPIStandardInvoice(std)
	if err != nil {
		return out, err
	}

	if err := out.FromBillingInvoiceStandard(stdAPI); err != nil {
		return out, fmt.Errorf("setting standard invoice union: %w", err)
	}

	return out, nil
}

// ToAPIChargeRealizationInvoice maps a realization's invoice to the charges
// API's own embedded invoice model: the standard invoice without the customer
// snapshot, which the embedding charge already identifies.
func ToAPIChargeRealizationInvoice(std billing.StandardInvoice) (apiv3models.BillingChargeRealizationInvoice, error) {
	stdAPI, err := toAPIStandardInvoice(std)
	if err != nil {
		return apiv3models.BillingChargeRealizationInvoice{}, err
	}

	return apiv3models.BillingChargeRealizationInvoice{
		CollectionAt:          stdAPI.CollectionAt,
		CreatedAt:             stdAPI.CreatedAt,
		Currency:              stdAPI.Currency,
		DeletedAt:             stdAPI.DeletedAt,
		Description:           stdAPI.Description,
		DraftUntil:            stdAPI.DraftUntil,
		DueAt:                 stdAPI.DueAt,
		ExternalReferences:    stdAPI.ExternalReferences,
		Id:                    stdAPI.Id,
		IssuedAt:              stdAPI.IssuedAt,
		Labels:                stdAPI.Labels,
		Number:                stdAPI.Number,
		QuantitySnapshottedAt: stdAPI.QuantitySnapshottedAt,
		SentToCustomerAt:      stdAPI.SentToCustomerAt,
		ServicePeriod:         stdAPI.ServicePeriod,
		Status:                stdAPI.Status,
		StatusDetails:         stdAPI.StatusDetails,
		Supplier:              stdAPI.Supplier,
		Totals:                stdAPI.Totals,
		Type:                  apiv3models.BillingChargeRealizationInvoiceTypeStandard,
		UpdatedAt:             stdAPI.UpdatedAt,
		ValidationIssues:      stdAPI.ValidationIssues,
		Workflow:              stdAPI.Workflow,
	}, nil
}

func toAPIStandardInvoice(std billing.StandardInvoice) (apiv3models.BillingInvoiceStandard, error) {
	// Sort lines for consistent output — matches v1 behavior.
	std.SortLines()

	workflow, err := toAPIWorkflow(std.Workflow)
	if err != nil {
		return apiv3models.BillingInvoiceStandard{}, fmt.Errorf("converting workflow: %w", err)
	}

	outLines, err := mapLines(std.Lines.OrEmpty())
	if err != nil {
		return apiv3models.BillingInvoiceStandard{}, err
	}

	return apiv3models.BillingInvoiceStandard{
		Id:                    std.ID,
		Number:                std.Number,
		Description:           std.Description,
		Labels:                labels.FromMetadata(std.Metadata),
		CreatedAt:             std.CreatedAt,
		UpdatedAt:             std.UpdatedAt,
		DeletedAt:             std.DeletedAt,
		IssuedAt:              std.IssuedAt,
		DueAt:                 std.DueAt,
		CollectionAt:          lo.ToPtr(lo.FromPtrOr(std.CollectionAt, std.CreatedAt)),
		DraftUntil:            std.DraftUntil,
		SentToCustomerAt:      std.SentToCustomerAt,
		QuantitySnapshottedAt: std.QuantitySnapshotedAt,
		ServicePeriod:         apiv3models.ClosedPeriod(lo.FromPtr(std.Period)),
		Currency:              apiv3models.CurrencyCode(std.Currency),
		Type:                  apiv3models.BillingInvoiceStandardTypeStandard,
		Status:                apiv3models.BillingInvoiceStandardStatus(std.Status.ShortStatus()),
		StatusDetails:         toAPIStatusDetails(std.StatusDetails, std.Status),
		Customer:              toAPIInvoiceCustomer(std.Customer),
		Supplier:              billingprofiles.ToAPIBillingSupplier(std.Supplier),
		Totals:                billingcommon.ToAPIBillingTotals(std.Totals),
		ValidationIssues:      billingcommon.ToAPIValidationIssues(std.ValidationIssues),
		ExternalReferences:    toAPIInvoiceExternalReferences(std.ExternalIDs),
		Workflow:              workflow,
		Lines:                 outLines,
	}, nil
}

func toAPIStatusDetails(d billing.StandardInvoiceStatusDetails, status billing.StandardInvoiceStatus) apiv3models.BillingInvoiceStatusDetails {
	return apiv3models.BillingInvoiceStatusDetails{
		Immutable:      d.Immutable,
		Failed:         d.Failed,
		ExtendedStatus: string(status),
		AvailableActions: apiv3models.BillingInvoiceAvailableActions{
			Advance:            toAPIActionDetails(d.AvailableActions.Advance),
			Approve:            toAPIActionDetails(d.AvailableActions.Approve),
			Delete:             toAPIActionDetails(d.AvailableActions.Delete),
			Retry:              toAPIActionDetails(d.AvailableActions.Retry),
			SnapshotQuantities: toAPIActionDetails(d.AvailableActions.SnapshotQuantities),
			// Void and Invoice actions are not exposed in v3.
		},
	}
}

func toAPIActionDetails(d *billing.StandardInvoiceAvailableActionDetails) *apiv3models.BillingInvoiceAvailableActionDetails {
	if d == nil {
		return nil
	}

	return &apiv3models.BillingInvoiceAvailableActionDetails{
		ResultingState: string(d.ResultingState),
	}
}

func toAPIInvoiceCustomer(c billing.InvoiceCustomer) apiv3models.BillingInvoiceCustomer {
	out := apiv3models.BillingInvoiceCustomer{
		Id:   c.CustomerID,
		Key:  c.Key,
		Name: c.Name,
	}

	if c.UsageAttribution != nil {
		out.UsageAttribution = &apiv3models.BillingCustomerUsageAttribution{
			SubjectKeys: c.UsageAttribution.SubjectKeys,
		}
	}

	if c.BillingAddress != nil && !lo.IsEmpty(*c.BillingAddress) {
		var country *apiv3models.CountryCode
		if c.BillingAddress.Country != nil {
			country = lo.ToPtr(apiv3models.CountryCode(*c.BillingAddress.Country))
		}

		out.BillingAddress = &apiv3models.Address{
			City:        c.BillingAddress.City,
			Country:     country,
			Line1:       c.BillingAddress.Line1,
			Line2:       c.BillingAddress.Line2,
			PhoneNumber: c.BillingAddress.PhoneNumber,
			PostalCode:  c.BillingAddress.PostalCode,
			State:       c.BillingAddress.State,
		}
	}

	return out
}

func toAPIWorkflow(w billing.InvoiceWorkflow) (apiv3models.BillingInvoiceWorkflowSettings, error) {
	config := w.Config

	invoicing := &apiv3models.BillingInvoiceWorkflowInvoicingSettings{
		AutoAdvance: lo.ToPtr(config.Invoicing.AutoAdvance),
		DraftPeriod: lo.ToPtr(config.Invoicing.DraftPeriod.String()),
		DueAfter:    lo.EmptyableToPtr(config.Invoicing.DueAfter.String()),
	}

	var payment *apiv3models.BillingWorkflowPaymentSettings
	switch config.Payment.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		p := apiv3models.BillingWorkflowPaymentSettings{}
		if err := p.FromBillingWorkflowPaymentChargeAutomaticallySettings(apiv3models.BillingWorkflowPaymentChargeAutomaticallySettings{
			CollectionMethod: "charge_automatically",
		}); err != nil {
			return apiv3models.BillingInvoiceWorkflowSettings{}, fmt.Errorf("converting payment settings: %w", err)
		}

		payment = &p
	case billing.CollectionMethodSendInvoice:
		p := apiv3models.BillingWorkflowPaymentSettings{}
		if err := p.FromBillingWorkflowPaymentSendInvoiceSettings(apiv3models.BillingWorkflowPaymentSendInvoiceSettings{
			CollectionMethod: "send_invoice",
			DueAfter:         lo.ToPtr(config.Invoicing.DueAfter.String()),
		}); err != nil {
			return apiv3models.BillingInvoiceWorkflowSettings{}, fmt.Errorf("converting payment settings: %w", err)
		}

		payment = &p
	}

	return apiv3models.BillingInvoiceWorkflowSettings{
		Apps: &apiv3models.BillingInvoiceWorkflowAppsReferences{
			Invoicing: apiv3models.BillingAppReference{
				Id: w.AppReferences.Invoicing.ID,
			},
			Payment: apiv3models.BillingAppReference{
				Id: w.AppReferences.Payment.ID,
			},
			Tax: apiv3models.BillingAppReference{
				Id: w.AppReferences.Tax.ID,
			},
		},
		SourceBillingProfile: apiv3models.BillingProfileReference{
			Id: w.SourceBillingProfileID,
		},
		Workflow: apiv3models.BillingInvoiceWorkflow{
			Invoicing: invoicing,
			Payment:   payment,
		},
	}, nil
}

func toAPIInvoiceExternalReferences(e externalid.InvoiceExternalIDs) *apiv3models.BillingInvoiceExternalReferences {
	if e.Invoicing == "" && e.Payment == "" {
		return nil
	}

	return &apiv3models.BillingInvoiceExternalReferences{
		InvoicingId: lo.EmptyableToPtr(e.Invoicing),
		PaymentId:   lo.EmptyableToPtr(e.Payment),
	}
}

func mapLines(lines []*billing.StandardLine) (*[]apiv3models.BillingInvoiceLine, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	out, err := lo.MapErr(lines, func(line *billing.StandardLine, _ int) (apiv3models.BillingInvoiceLine, error) {
		mapped, err := mapStandardLine(line)
		if err != nil {
			return apiv3models.BillingInvoiceLine{}, fmt.Errorf("mapping line[%s]: %w", line.ID, err)
		}

		var invoiceLine apiv3models.BillingInvoiceLine
		if err := invoiceLine.FromBillingInvoiceStandardLine(mapped); err != nil {
			return apiv3models.BillingInvoiceLine{}, fmt.Errorf("setting standard line union: %w", err)
		}

		return invoiceLine, nil
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func mapStandardLine(line *billing.StandardLine) (apiv3models.BillingInvoiceStandardLine, error) {
	rateCard, err := mapRateCard(line)
	if err != nil {
		return apiv3models.BillingInvoiceStandardLine{}, fmt.Errorf("mapping rate card: %w", err)
	}

	line.SortDetailedLines()
	detailedLines, err := mapDetailedLines(line.DetailedLines)
	if err != nil {
		return apiv3models.BillingInvoiceStandardLine{}, fmt.Errorf("mapping detailed lines: %w", err)
	}

	var chargeRef *apiv3models.BillingChargeReference
	if line.ChargeID != nil {
		chargeRef = &apiv3models.BillingChargeReference{Id: *line.ChargeID}
	}

	var subRef *apiv3models.BillingSubscriptionReference
	if line.Subscription != nil {
		subRef = lo.ToPtr(billingcommon.ConvertSubscriptionRefToAPI(meta.SubscriptionReference{
			SubscriptionID: line.Subscription.SubscriptionID,
			PhaseID:        line.Subscription.PhaseID,
			ItemID:         line.Subscription.ItemID,
		}))
	}

	return apiv3models.BillingInvoiceStandardLine{
		Id:                  lo.ToPtr(line.ID),
		Name:                line.Name,
		Description:         line.Description,
		Labels:              labels.FromMetadata(line.Metadata),
		CreatedAt:           line.CreatedAt,
		UpdatedAt:           line.UpdatedAt,
		DeletedAt:           line.DeletedAt,
		Type:                apiv3models.BillingInvoiceStandardLineTypeStandardLine,
		LifecycleController: billingcommon.ConvertLifecycleControllerToAPI(line.ManagedBy),
		ServicePeriod:       billingcommon.ConvertClosedPeriodToAPI(line.Period),
		Totals:              billingcommon.ToAPIBillingTotals(line.Totals),
		Charge:              chargeRef,
		Subscription:        subRef,
		ExternalReferences:  toAPILineExternalReferences(line.ExternalIDs),
		CreditsApplied:      mapCreditApplies(line.CreditsApplied),
		Discounts:           mapLineDiscounts(line.Discounts),
		RateCard:            rateCard,
		DetailedLines:       detailedLines,
		UsageQuantityDetail: toAPIUsageQuantityDetail(line),
	}, nil
}

// toAPIUsageQuantityDetail surfaces the UnitConfig quantity audit trail for a
// standard line: the raw metered quantity, the net invoiced quantity, and the
// configured display unit. The config itself is not repeated here -- it stays on
// rate_card.unit_config. It returns nil for lines without unit conversion so
// existing invoices are byte-identical: non-usage-based lines and the charges
// custom-currency overage path (synthetic metered quantity, no config) both
// have a nil UnitConfig and emit nothing.
func toAPIUsageQuantityDetail(line *billing.StandardLine) *apiv3models.BillingInvoiceUsageQuantityDetail {
	uc := line.GetUnitConfig()
	if line.UsageBased == nil || uc == nil {
		return nil
	}

	// The metered and billed quantities are pointer-typed and stay nil on lines
	// that never rated; without both endpoints there is no trail to surface.
	if line.UsageBased.MeteredQuantity == nil || line.UsageBased.Quantity == nil {
		return nil
	}

	return &apiv3models.BillingInvoiceUsageQuantityDetail{
		RawQuantity:      line.UsageBased.MeteredQuantity.String(),
		InvoicedQuantity: line.UsageBased.Quantity.String(),
		DisplayUnit:      uc.DisplayUnit,
	}
}

func mapRateCard(line *billing.StandardLine) (apiv3models.BillingInvoiceLineRateCard, error) {
	if line.UsageBased == nil {
		return apiv3models.BillingInvoiceLineRateCard{}, fmt.Errorf("standard line %s has no usage-based configuration", line.ID)
	}

	if line.UsageBased.Price == nil {
		return apiv3models.BillingInvoiceLineRateCard{}, fmt.Errorf("standard line %s has no price set", line.ID)
	}

	price, err := plans.ToAPIBillingPrice(line.UsageBased.Price)
	if err != nil {
		return apiv3models.BillingInvoiceLineRateCard{}, fmt.Errorf("mapping price: %w", err)
	}

	rc := apiv3models.BillingInvoiceLineRateCard{
		Price:      price,
		FeatureKey: lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		Discounts:  toAPIRateCardDiscounts(line.RateCardDiscounts),
		TaxConfig:  tax.ToAPITaxCodeConfig(line.TaxConfig),
	}

	if uc := line.GetUnitConfig(); uc != nil {
		rc.UnitConfig = lo.ToPtr(plans.ToAPIBillingUnitConfig(*uc))
	}

	return rc, nil
}

func mapDetailedLines(dls billing.DetailedLines) ([]apiv3models.BillingInvoiceDetailedLine, error) {
	if len(dls) == 0 {
		return lo.Empty[[]apiv3models.BillingInvoiceDetailedLine](), nil
	}

	return lo.MapErr(dls, func(dl billing.DetailedLine, _ int) (apiv3models.BillingInvoiceDetailedLine, error) {
		mapped, err := mapDetailedLine(dl)
		if err != nil {
			return apiv3models.BillingInvoiceDetailedLine{}, fmt.Errorf("mapping detailed line[%s]: %w", dl.ID, err)
		}

		return mapped, nil
	})
}

func mapDetailedLine(dl billing.DetailedLine) (apiv3models.BillingInvoiceDetailedLine, error) {
	return apiv3models.BillingInvoiceDetailedLine{
		Id:                 dl.ID,
		Name:               dl.Name,
		Description:        dl.Description,
		CreatedAt:          dl.CreatedAt,
		UpdatedAt:          dl.UpdatedAt,
		DeletedAt:          dl.DeletedAt,
		Category:           apiv3models.BillingInvoiceDetailedLineCostCategory(dl.Category),
		ServicePeriod:      billingcommon.ConvertClosedPeriodToAPI(dl.ServicePeriod),
		Quantity:           dl.Quantity.String(),
		UnitPrice:          dl.PerUnitAmount.String(),
		Totals:             billingcommon.ToAPIBillingTotals(dl.Totals),
		CreditsApplied:     mapCreditApplies(dl.CreditsApplied),
		Discounts:          mapAmountDiscounts(dl.AmountDiscounts),
		ExternalReferences: toAPILineExternalReferences(dl.ExternalIDs),
	}, nil
}

func toAPIRateCardDiscounts(d billing.Discounts) *apiv3models.BillingRateCardDiscounts {
	if d.IsEmpty() {
		return nil
	}

	result := &apiv3models.BillingRateCardDiscounts{}

	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}

	if d.Usage != nil {
		s := d.Usage.Quantity.String()
		result.Usage = &s
	}

	return result
}

func mapCreditApplies(ca creditsapplied.CreditsApplied) *[]apiv3models.BillingInvoiceLineCreditsApplied {
	if len(ca) == 0 {
		return nil
	}

	out := lo.Map(ca, func(c creditsapplied.CreditApplied, _ int) apiv3models.BillingInvoiceLineCreditsApplied {
		return apiv3models.BillingInvoiceLineCreditsApplied{
			Amount:      c.Amount.String(),
			Description: lo.EmptyableToPtr(c.Description),
		}
	})

	return &out
}

// mapLineDiscounts maps usage discounts from a parent StandardLine (amount discounts live on
// child detailed lines and are mapped separately via mapAmountDiscounts).
func mapLineDiscounts(d billing.StandardLineDiscounts) *apiv3models.BillingInvoiceLineDiscounts {
	if len(d.Usage) == 0 {
		return nil
	}

	usageDiscounts := lo.Map(d.Usage, func(ud billing.UsageLineDiscountManaged, _ int) apiv3models.BillingInvoiceLineUsageDiscount {
		return apiv3models.BillingInvoiceLineUsageDiscount{
			Id:                 ud.ID,
			Quantity:           ud.Quantity.String(),
			Description:        ud.Description,
			ExternalReferences: toAPILineExternalReferences(ud.ExternalIDs),
			Reason:             apiv3models.BillingInvoiceDiscountReason(ud.Reason.Type()),
		}
	})

	return &apiv3models.BillingInvoiceLineDiscounts{
		Usage: &usageDiscounts,
	}
}

func mapAmountDiscounts(d billing.AmountLineDiscountsManaged) *apiv3models.BillingInvoiceLineDiscounts {
	if len(d) == 0 {
		return nil
	}

	amountDiscounts := lo.Map(d, func(ad billing.AmountLineDiscountManaged, _ int) apiv3models.BillingInvoiceLineAmountDiscount {
		return apiv3models.BillingInvoiceLineAmountDiscount{
			Id:                 ad.ID,
			Amount:             ad.Amount.String(),
			Description:        ad.Description,
			ExternalReferences: toAPILineExternalReferences(ad.ExternalIDs),
			Reason:             apiv3models.BillingInvoiceDiscountReason(ad.Reason.Type()),
		}
	})

	return &apiv3models.BillingInvoiceLineDiscounts{
		Amount: &amountDiscounts,
	}
}

func toAPILineExternalReferences(e externalid.LineExternalIDs) *apiv3models.BillingInvoiceLineExternalReferences {
	if e.Invoicing == "" {
		return nil
	}

	return &apiv3models.BillingInvoiceLineExternalReferences{
		InvoicingId: lo.ToPtr(e.Invoicing),
	}
}

// invoiceSortField maps v3 sort field names to the billing domain's InvoiceOrderBy values.
func FromAPIInvoiceSortField(ctx context.Context, field string) (apiv1models.InvoiceOrderBy, error) {
	switch field {
	case "issued_at":
		return apiv1models.InvoiceOrderByIssuedAt, nil
	case "created_at":
		return apiv1models.InvoiceOrderByCreatedAt, nil
	case "service_period_start":
		return apiv1models.InvoiceOrderByPeriodStart, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "issued_at", "created_at", "service_period_start")
	}
}

// mergeStandardInvoiceFromAPI applies the mutable fields of an update request onto an
// already-loaded standard invoice, in place. Only description, labels, supplier, customer,
// workflow, and top-level lines are mutable; all other invoice fields are left untouched.
func mergeStandardInvoiceFromAPI(inv *billing.StandardInvoice, req apiv3models.UpdateInvoiceStandardRequest) error {
	inv.Description = req.Description

	metadata, err := labels.ToMetadata(req.Labels)
	if err != nil {
		return fmt.Errorf("converting labels: %w", err)
	}

	inv.Metadata = metadata

	inv.Supplier = mergeInvoiceSupplierFromAPI(inv.Supplier, req.Supplier)
	inv.Customer = mergeInvoiceCustomerFromAPI(inv.Customer, req.Customer)

	workflow, err := mergeInvoiceWorkflowFromAPI(inv.Workflow, req.Workflow)
	if err != nil {
		return fmt.Errorf("merging workflow: %w", err)
	}

	inv.Workflow = workflow

	lines, err := mergeStandardInvoiceLinesFromAPI(inv, req.Lines)
	if err != nil {
		return fmt.Errorf("merging lines: %w", err)
	}

	inv.Lines = lines

	return nil
}

// mergeInvoiceSupplierFromAPI applies the editable BillingSupplier snapshot fields (name, tax
// id, address) onto the existing supplier contact. The party ID is not part of the update
// request and is left untouched.
func mergeInvoiceSupplierFromAPI(existing billing.SupplierContact, updated apiv3models.UpdateSupplier) billing.SupplierContact {
	existing.Name = lo.FromPtrOr(updated.Name, "")

	if updated.Addresses != nil {
		existing.Address = billingprofiles.FromAPIAddress(apiv3models.Address(updated.Addresses.BillingAddress))
	} else {
		existing.Address = models.Address{}
	}

	if updated.TaxId != nil {
		existing.TaxCode = updated.TaxId.Code
	} else {
		existing.TaxCode = nil
	}

	return existing
}

// mergeInvoiceCustomerFromAPI applies the editable customer snapshot fields (name, billing
// address) onto the existing customer snapshot. CustomerID, key, and usage attribution are
// immutable identity fields tied to the customer at invoice creation time and are left
// untouched, regardless of what the request echoes back for them.
func mergeInvoiceCustomerFromAPI(existing billing.InvoiceCustomer, updated apiv3models.UpdateInvoiceCustomer) billing.InvoiceCustomer {
	existing.Name = updated.Name

	if updated.BillingAddress != nil {
		addr := billingprofiles.FromAPIAddress(apiv3models.Address(*updated.BillingAddress))
		existing.BillingAddress = &addr
	} else {
		existing.BillingAddress = nil
	}

	return existing
}

// mergeInvoiceWorkflowFromAPI applies the editable per-invoice workflow settings (invoicing
// auto-advance/draft period, payment collection method/due date) onto the existing workflow
// config. Omitting the invoicing or payment sub-object leaves that part of the config
// unchanged; omitted fields within a provided sub-object fall back to
// billing.DefaultWorkflowConfig, mirroring the v1 replace-update semantics.
func mergeInvoiceWorkflowFromAPI(existing billing.InvoiceWorkflow, updated apiv3models.UpdateInvoiceWorkflowSettings) (billing.InvoiceWorkflow, error) {
	if invoicing := updated.Workflow.Invoicing; invoicing != nil {
		existing.Config.Invoicing.AutoAdvance = lo.FromPtrOr(invoicing.AutoAdvance, billing.DefaultWorkflowConfig.Invoicing.AutoAdvance)

		if invoicing.DraftPeriod == nil {
			existing.Config.Invoicing.DraftPeriod = billing.DefaultWorkflowConfig.Invoicing.DraftPeriod
		} else {
			period, err := datetime.ISODurationString(*invoicing.DraftPeriod).Parse()
			if err != nil {
				return existing, billing.ValidationError{Err: fmt.Errorf("failed to parse draft period: %w", err)}
			}

			existing.Config.Invoicing.DraftPeriod = period
		}
	}

	if payment := updated.Workflow.Payment; payment != nil {
		disc, err := payment.Discriminator()
		if err != nil {
			return existing, billing.ValidationError{Err: fmt.Errorf("failed to read payment settings type: %w", err)}
		}

		switch disc {
		case "charge_automatically":
			existing.Config.Payment.CollectionMethod = billing.CollectionMethodChargeAutomatically
			existing.Config.Invoicing.DueAfter = billing.DefaultWorkflowConfig.Invoicing.DueAfter
		case "send_invoice":
			sendInvoice, err := payment.AsUpdateBillingWorkflowPaymentSendInvoiceSettings()
			if err != nil {
				return existing, billing.ValidationError{Err: fmt.Errorf("reading send invoice settings: %w", err)}
			}

			existing.Config.Payment.CollectionMethod = billing.CollectionMethodSendInvoice

			if sendInvoice.DueAfter == nil {
				existing.Config.Invoicing.DueAfter = billing.DefaultWorkflowConfig.Invoicing.DueAfter
			} else {
				period, err := datetime.ISODurationString(*sendInvoice.DueAfter).Parse()
				if err != nil {
					return existing, billing.ValidationError{Err: fmt.Errorf("failed to parse due after: %w", err)}
				}

				existing.Config.Invoicing.DueAfter = period
			}
		default:
			return existing, billing.ValidationError{Err: fmt.Errorf("unsupported payment collection method: %s", disc)}
		}
	}

	return existing, nil
}

// mergeStandardInvoiceLinesFromAPI reconciles the invoice's top-level lines against the
// update request: lines matched by ID are merged in place, lines without an ID (or with an
// unknown ID) are created, and existing lines omitted from the request are tombstoned. A nil
// lines pointer leaves the invoice's lines untouched, since it means the field wasn't sent.
func mergeStandardInvoiceLinesFromAPI(inv *billing.StandardInvoice, lines *[]apiv3models.UpdateInvoiceLine) (billing.StandardInvoiceLines, error) {
	if lines == nil {
		return inv.Lines, nil
	}

	linesByID, _ := slicesx.UniqueGroupBy(inv.Lines.OrEmpty(), func(line *billing.StandardLine) string {
		return line.ID
	})

	foundLines := set.New[string]()
	out := make([]*billing.StandardLine, 0, len(*lines))

	processedIDs := set.New[string]()

	for _, apiLine := range *lines {
		stdLine, err := apiLine.AsUpdateInvoiceStandardLine()
		if err != nil {
			return billing.StandardInvoiceLines{}, fmt.Errorf("reading line: %w", err)
		}

		id := lo.FromPtr(stdLine.Id)

		if id != "" {
			if processedIDs.Has(id) {
				return billing.StandardInvoiceLines{}, billing.ValidationError{
					Err: fmt.Errorf("duplicate line ID %q in request", id),
				}
			}

			processedIDs.Add(id)
		}

		existingLine, existingLineFound := linesByID[id]

		if id == "" || !existingLineFound {
			// We allow injecting fake IDs for new lines, so that discounts can reference
			// those, but we are not persisting them to the database.
			newLine, err := standardLineFromAPI(stdLine, inv)
			if err != nil {
				return billing.StandardInvoiceLines{}, fmt.Errorf("creating line: %w", err)
			}

			out = append(out, newLine)
		} else {
			foundLines.Add(id)

			mergedLine, err := mergeStandardLineFromAPI(existingLine, stdLine)
			if err != nil {
				return billing.StandardInvoiceLines{}, fmt.Errorf("merging line[%s]: %w", id, err)
			}

			out = append(out, mergedLine)
		}
	}

	lineIDs := set.New(lo.Keys(linesByID)...)

	deletedLines := set.Subtract(lineIDs, foundLines).AsSlice()
	for _, id := range deletedLines {
		existingLine := linesByID[id]
		existingLine.DeletedAt = lo.ToPtr(clock.Now())
		out = append(out, existingLine)
	}

	return billing.NewStandardInvoiceLines(out), nil
}

// standardLineFromAPI builds a new top-level standard line from an update request line that
// has no matching existing line (empty or unrecognized ID).
func standardLineFromAPI(line apiv3models.UpdateInvoiceStandardLine, inv *billing.StandardInvoice) (*billing.StandardLine, error) {
	price, taxConfig, featureKey, desiredDiscounts, err := mapRateCardFromAPI(line.RateCard)
	if err != nil {
		return nil, fmt.Errorf("mapping rate card: %w", err)
	}

	metadata, err := labels.ToMetadata(line.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
	}

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   inv.Namespace,
				Name:        line.Name,
				Description: line.Description,
			}),

			Metadata: metadata,

			InvoiceID: inv.ID,
			Currency:  inv.Currency,

			Period: timeutil.ClosedPeriod{
				From: line.ServicePeriod.From.Truncate(streaming.MinimumWindowSizeDuration),
				To:   line.ServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration),
			},
			// InvoiceAt has no scheduling meaning for a manually added standard line (it
			// only carries the original gathering-line timestamp for lines rendered from
			// gathering invoices, per StandardLineBase.InvoiceAt's doc comment) but is
			// required to be non-zero. Use the creation timestamp as the sanctioned fallback.
			InvoiceAt: clock.Now().Truncate(streaming.MinimumWindowSizeDuration),

			TaxConfig:         taxConfig,
			RateCardDiscounts: billing.DiscountsFromProductCatalog(desiredDiscounts),
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      price,
			FeatureKey: featureKey,
		},
	}, nil
}

// mergeStandardLineFromAPI applies the editable fields of an update request line onto an
// existing top-level standard line, matched by ID.
func mergeStandardLineFromAPI(existing *billing.StandardLine, line apiv3models.UpdateInvoiceStandardLine) (*billing.StandardLine, error) {
	price, taxConfig, featureKey, desiredDiscounts, err := mapRateCardFromAPI(line.RateCard)
	if err != nil {
		return nil, fmt.Errorf("mapping rate card: %w", err)
	}

	metadata, err := labels.ToMetadata(line.Labels)
	if err != nil {
		return nil, fmt.Errorf("converting labels: %w", err)
	}

	existing.Metadata = metadata
	existing.Name = line.Name
	existing.Description = line.Description

	existing.Period.From = line.ServicePeriod.From.Truncate(streaming.MinimumWindowSizeDuration)
	existing.Period.To = line.ServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)

	existing.TaxConfig = taxConfig
	existing.RateCardDiscounts = existing.RateCardDiscounts.ReplaceFromProductCatalog(desiredDiscounts)
	if existing.UsageBased == nil {
		return nil, fmt.Errorf("existing line %s has no usage-based pricing", existing.ID)
	}

	existing.UsageBased.Price = price
	existing.UsageBased.FeatureKey = featureKey

	return existing, nil
}

// mapRateCardFromAPI maps an update request's rate card onto its domain price, tax config,
// feature key, and discounts. Feature key requiredness relative to the price type is enforced
// by billing.UsageBasedLine.Validate downstream, not here.
func mapRateCardFromAPI(rc apiv3models.UpdateInvoiceLineRateCard) (*productcatalog.Price, *productcatalog.TaxCodeConfig, string, productcatalog.Discounts, error) {
	price, err := plans.FromAPIBillingPrice(apiv3models.BillingPrice(rc.Price), nil)
	if err != nil {
		return nil, nil, "", productcatalog.Discounts{}, fmt.Errorf("mapping price: %w", err)
	}

	var discounts productcatalog.Discounts
	if rc.Discounts != nil {
		pcDiscounts, err := plans.FromAPIBillingRateCardDiscounts(apiv3models.BillingRateCardDiscounts(*rc.Discounts))
		if err != nil {
			return nil, nil, "", productcatalog.Discounts{}, fmt.Errorf("mapping discounts: %w", err)
		}

		discounts = pcDiscounts
	}

	// The schema allows tax_config with neither code nor behavior; the at-least-one-of
	// rule is enforced here, and an explicit code must carry an id. The billing validation
	// error wrap makes the route's error encoder respond with a 400 instead of the 500
	// fallback.
	pcTaxConfig, err := tax.FromAPITaxCodeConfig(fromAPIUpdateTaxCodeConfig(rc.TaxConfig))
	if err != nil {
		return nil, nil, "", productcatalog.Discounts{}, billing.ValidationError{
			Err: fmt.Errorf("mapping tax config: %w", err),
		}
	}

	return price, pcTaxConfig, lo.FromPtrOr(rc.FeatureKey, ""), discounts, nil
}

func fromAPIUpdateTaxCodeConfig(taxConfig *apiv3models.UpdateTaxCodeConfig) *apiv3models.TaxCodeConfig {
	if taxConfig == nil {
		return nil
	}

	result := &apiv3models.TaxCodeConfig{
		Behavior: taxConfig.Behavior,
	}

	if taxConfig.Code != nil {
		result.Code = &apiv3models.TaxCodeReference{
			Id: taxConfig.Code.Id,
		}
	}

	return result
}
