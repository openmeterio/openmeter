package billinginvoices

import (
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/addons"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	chargeshandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/charges"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ToAPIBillingInvoice converts a billing.Invoice domain union to the v3 API type.
func ToAPIBillingInvoice(inv billing.Invoice) (api.BillingInvoice, error) {
	var out api.BillingInvoice

	switch inv.Type() {
	case billing.InvoiceTypeStandard:
		std, err := inv.AsStandardInvoice()
		if err != nil {
			return out, fmt.Errorf("reading standard invoice: %w", err)
		}

		stdAPI, err := toAPIStandardInvoice(std)
		if err != nil {
			return out, err
		}

		if err := out.FromBillingStandardInvoice(stdAPI); err != nil {
			return out, fmt.Errorf("setting standard invoice union: %w", err)
		}

	default:
		return out, models.NewGenericNotFoundError(fmt.Errorf("unsupported invoice type %q", inv.Type()))
	}

	return out, nil
}

func toAPIStandardInvoice(std billing.StandardInvoice) (api.BillingStandardInvoice, error) {
	// Sort lines for consistent output — matches v1 behavior.
	std.SortLines()

	workflow, err := toAPIWorkflow(std.Workflow)
	if err != nil {
		return api.BillingStandardInvoice{}, fmt.Errorf("converting workflow: %w", err)
	}

	outLines, err := mapLines(std.Lines.OrEmpty())
	if err != nil {
		return api.BillingStandardInvoice{}, err
	}

	return api.BillingStandardInvoice{
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
		ServicePeriod:         api.ClosedPeriod(lo.FromPtr(std.Period)),
		Currency:              api.CurrencyCode(std.Currency),
		Type:                  api.BillingStandardInvoiceTypeStandard,
		Status:                api.BillingStandardInvoiceStatus(std.Status.ShortStatus()),
		StatusDetails:         toAPIStatusDetails(std.StatusDetails, std.Status),
		Customer:              toAPIInvoiceCustomer(std.Customer),
		Supplier:              billingprofiles.ToAPIBillingSupplier(std.Supplier),
		Totals:                chargeshandler.ToAPIBillingTotals(std.Totals),
		ValidationIssues:      mapValidationIssues(std.ValidationIssues),
		ExternalIds:           toAPIInvoiceExternalIds(std.ExternalIDs),
		Workflow:              workflow,
		Lines:                 outLines,
	}, nil
}

func toAPIStatusDetails(d billing.StandardInvoiceStatusDetails, status billing.StandardInvoiceStatus) api.BillingInvoiceStatusDetails {
	return api.BillingInvoiceStatusDetails{
		Immutable:      d.Immutable,
		Failed:         d.Failed,
		ExtendedStatus: string(status),
		AvailableActions: api.BillingInvoiceAvailableActions{
			Advance:            toAPIActionDetails(d.AvailableActions.Advance),
			Approve:            toAPIActionDetails(d.AvailableActions.Approve),
			Delete:             toAPIActionDetails(d.AvailableActions.Delete),
			Retry:              toAPIActionDetails(d.AvailableActions.Retry),
			SnapshotQuantities: toAPIActionDetails(d.AvailableActions.SnapshotQuantities),
			// Void and Invoice actions are not exposed in v3.
		},
	}
}

func toAPIActionDetails(d *billing.StandardInvoiceAvailableActionDetails) *api.BillingInvoiceAvailableActionDetails {
	if d == nil {
		return nil
	}

	return &api.BillingInvoiceAvailableActionDetails{
		ResultingState: string(d.ResultingState),
	}
}

func toAPIInvoiceCustomer(c billing.InvoiceCustomer) api.BillingInvoiceCustomer {
	out := api.BillingInvoiceCustomer{
		Id:   c.CustomerID,
		Key:  c.Key,
		Name: c.Name,
		UsageAttribution: api.BillingCustomerUsageAttribution{
			SubjectKeys: []api.UsageAttributionSubjectKey{},
		},
	}

	if c.UsageAttribution != nil {
		out.UsageAttribution.SubjectKeys = c.UsageAttribution.SubjectKeys
	}

	if c.BillingAddress != nil && !lo.IsEmpty(*c.BillingAddress) {
		var country *api.CountryCode
		if c.BillingAddress.Country != nil {
			country = lo.ToPtr(api.CountryCode(*c.BillingAddress.Country))
		}
		out.BillingAddress = &api.BillingAddress{
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

func toAPIWorkflow(w billing.InvoiceWorkflow) (api.BillingInvoiceWorkflowSettings, error) {
	config := w.Config

	invoicing := &api.BillingInvoiceWorkflowInvoicingSettings{
		AutoAdvance: lo.ToPtr(config.Invoicing.AutoAdvance),
		DraftPeriod: lo.ToPtr(config.Invoicing.DraftPeriod.String()),
	}

	var payment *api.BillingWorkflowPaymentSettings
	switch config.Payment.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		p := api.BillingWorkflowPaymentSettings{}
		if err := p.FromBillingWorkflowPaymentChargeAutomaticallySettings(api.BillingWorkflowPaymentChargeAutomaticallySettings{
			CollectionMethod: "charge_automatically",
		}); err != nil {
			return api.BillingInvoiceWorkflowSettings{}, fmt.Errorf("converting payment settings: %w", err)
		}
		payment = &p
	case billing.CollectionMethodSendInvoice:
		p := api.BillingWorkflowPaymentSettings{}
		if err := p.FromBillingWorkflowPaymentSendInvoiceSettings(api.BillingWorkflowPaymentSendInvoiceSettings{
			CollectionMethod: "send_invoice",
			DueAfter:         lo.ToPtr(config.Invoicing.DueAfter.String()),
		}); err != nil {
			return api.BillingInvoiceWorkflowSettings{}, fmt.Errorf("converting payment settings: %w", err)
		}
		payment = &p
	}

	return api.BillingInvoiceWorkflowSettings{
		SourceBillingProfileId: w.SourceBillingProfileID,
		Workflow: api.BillingInvoiceWorkflow{
			Invoicing: invoicing,
			Payment:   payment,
		},
	}, nil
}

func mapValidationIssues(issues []billing.ValidationIssue) *[]api.BillingInvoiceValidationIssue {
	if len(issues) == 0 {
		return nil
	}

	out := lo.Map(issues, func(v billing.ValidationIssue, _ int) api.BillingInvoiceValidationIssue {
		return api.BillingInvoiceValidationIssue{
			Severity: api.BillingInvoiceValidationIssueSeverity(v.Severity),
			Message:  v.Message,
			Code:     v.Code,
			Field:    lo.EmptyableToPtr(v.Path),
		}
	})

	return &out
}

func toAPIInvoiceExternalIds(e externalid.InvoiceExternalIDs) *api.BillingInvoiceExternalIds {
	if e.Invoicing == "" {
		return nil
	}

	return &api.BillingInvoiceExternalIds{
		Invoicing: lo.ToPtr(e.Invoicing),
	}
}

func mapLines(lines []*billing.StandardLine) (*[]api.BillingInvoiceLine, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	out, err := lo.MapErr(lines, func(line *billing.StandardLine, _ int) (api.BillingInvoiceLine, error) {
		mapped, err := mapStandardLine(line)
		if err != nil {
			return api.BillingInvoiceLine{}, fmt.Errorf("mapping line[%s]: %w", line.ID, err)
		}

		var invoiceLine api.BillingInvoiceLine
		invoiceLine.FromBillingInvoiceStandardLine(mapped)

		return invoiceLine, nil
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func mapStandardLine(line *billing.StandardLine) (api.BillingInvoiceStandardLine, error) {
	rateCard, err := mapRateCard(line)
	if err != nil {
		return api.BillingInvoiceStandardLine{}, fmt.Errorf("mapping rate card: %w", err)
	}

	line.SortDetailedLines()
	detailedLines, err := mapDetailedLines(line.DetailedLines)
	if err != nil {
		return api.BillingInvoiceStandardLine{}, fmt.Errorf("mapping detailed lines: %w", err)
	}

	var chargeRef *api.BillingChargeReference
	if line.ChargeID != nil {
		chargeRef = &api.BillingChargeReference{Id: *line.ChargeID}
	}

	var subRef *api.BillingSubscriptionReference
	if line.Subscription != nil {
		subRef = lo.ToPtr(chargeshandler.ConvertSubscriptionRefToAPI(meta.SubscriptionReference{
			SubscriptionID: line.Subscription.SubscriptionID,
			PhaseID:        line.Subscription.PhaseID,
			ItemID:         line.Subscription.ItemID,
		}))
	}

	return api.BillingInvoiceStandardLine{
		Id:             line.ID,
		Name:           line.Name,
		Description:    line.Description,
		Labels:         labels.FromMetadata(line.Metadata),
		CreatedAt:      line.CreatedAt,
		UpdatedAt:      line.UpdatedAt,
		DeletedAt:      line.DeletedAt,
		Type:           api.BillingInvoiceStandardLineTypeStandardLine,
		ManagedBy:      api.BillingInvoiceLineManagedBy(line.ManagedBy),
		ServicePeriod:  chargeshandler.ConvertClosedPeriodToAPI(line.Period),
		Totals:         chargeshandler.ToAPIBillingTotals(line.Totals),
		Charge:         chargeRef,
		Subscription:   subRef,
		ExternalIds:    toAPILineExternalIds(line.ExternalIDs),
		CreditsApplied: mapCreditApplies(line.CreditsApplied),
		Discounts:      mapLineDiscounts(line.Discounts),
		RateCard:       rateCard,
		DetailedLines:  detailedLines,
	}, nil
}

func mapRateCard(line *billing.StandardLine) (api.BillingInvoiceLineRateCard, error) {
	if line.UsageBased == nil {
		return api.BillingInvoiceLineRateCard{}, nil
	}

	if line.UsageBased.Price == nil {
		return api.BillingInvoiceLineRateCard{}, nil
	}

	price, err := plans.ToAPIBillingPrice(line.UsageBased.Price)
	if err != nil {
		return api.BillingInvoiceLineRateCard{}, fmt.Errorf("mapping price: %w", err)
	}

	rc := api.BillingInvoiceLineRateCard{
		Price:      price,
		FeatureKey: lo.EmptyableToPtr(line.UsageBased.FeatureKey),
		Discounts:  toAPIRateCardDiscounts(line.RateCardDiscounts),
		TaxConfig:  addons.ToAPIBillingRateCardTaxConfig(line.TaxConfig.ToProductCatalog()),
	}

	return rc, nil
}

func mapDetailedLines(dls billing.DetailedLines) ([]api.BillingInvoiceDetailedLine, error) {
	if len(dls) == 0 {
		return lo.Empty[[]api.BillingInvoiceDetailedLine](), nil
	}

	return lo.MapErr(dls, func(dl billing.DetailedLine, _ int) (api.BillingInvoiceDetailedLine, error) {
		mapped, err := mapDetailedLine(dl)
		if err != nil {
			return api.BillingInvoiceDetailedLine{}, fmt.Errorf("mapping detailed line[%s]: %w", dl.ID, err)
		}
		return mapped, nil
	})
}

func mapDetailedLine(dl billing.DetailedLine) (api.BillingInvoiceDetailedLine, error) {
	return api.BillingInvoiceDetailedLine{
		Id:             dl.ID,
		Name:           dl.Name,
		Description:    dl.Description,
		CreatedAt:      dl.CreatedAt,
		UpdatedAt:      dl.UpdatedAt,
		DeletedAt:      dl.DeletedAt,
		Category:       api.BillingInvoiceDetailedLineCostCategory(dl.Category),
		ServicePeriod:  chargeshandler.ConvertClosedPeriodToAPI(dl.ServicePeriod),
		Quantity:       dl.Quantity.String(),
		UnitPrice:      dl.PerUnitAmount.String(),
		Totals:         chargeshandler.ToAPIBillingTotals(dl.Totals),
		CreditsApplied: mapCreditApplies(dl.CreditsApplied),
		Discounts:      mapAmountDiscounts(dl.AmountDiscounts),
		ExternalIds:    toAPILineExternalIds(dl.ExternalIDs),
	}, nil
}

func toAPIRateCardDiscounts(d billing.Discounts) *api.BillingRateCardDiscounts {
	if d.IsEmpty() {
		return nil
	}

	result := &api.BillingRateCardDiscounts{}

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

func mapCreditApplies(ca creditsapplied.CreditsApplied) *[]api.BillingInvoiceLineCreditsApplied {
	if len(ca) == 0 {
		return nil
	}

	out := lo.Map(ca, func(c creditsapplied.CreditApplied, _ int) api.BillingInvoiceLineCreditsApplied {
		return api.BillingInvoiceLineCreditsApplied{
			Amount:      c.Amount.String(),
			Description: lo.EmptyableToPtr(c.Description),
		}
	})

	return &out
}

// mapLineDiscounts maps usage discounts from a parent StandardLine (amount discounts live on
// child detailed lines and are mapped separately via mapAmountDiscounts).
func mapLineDiscounts(d billing.StandardLineDiscounts) *api.BillingInvoiceLineDiscounts {
	if len(d.Usage) == 0 {
		return nil
	}

	usageDiscounts := lo.Map(d.Usage, func(ud billing.UsageLineDiscountManaged, _ int) api.BillingInvoiceLineUsageDiscount {
		return api.BillingInvoiceLineUsageDiscount{
			Id:          ud.ID,
			Quantity:    ud.Quantity.String(),
			Description: ud.Description,
			ExternalIds: toAPILineExternalIds(ud.ExternalIDs),
			Reason:      api.BillingInvoiceDiscountReason(ud.Reason.Type()),
		}
	})

	return &api.BillingInvoiceLineDiscounts{
		Usage: &usageDiscounts,
	}
}

func mapAmountDiscounts(d billing.AmountLineDiscountsManaged) *api.BillingInvoiceLineDiscounts {
	if len(d) == 0 {
		return nil
	}

	amountDiscounts := lo.Map(d, func(ad billing.AmountLineDiscountManaged, _ int) api.BillingInvoiceLineAmountDiscount {
		return api.BillingInvoiceLineAmountDiscount{
			Id:          ad.ID,
			Amount:      ad.Amount.String(),
			Description: ad.Description,
			ExternalIds: toAPILineExternalIds(ad.ExternalIDs),
			Reason:      api.BillingInvoiceDiscountReason(ad.Reason.Type()),
		}
	})

	return &api.BillingInvoiceLineDiscounts{
		Amount: &amountDiscounts,
	}
}

func toAPILineExternalIds(e externalid.LineExternalIDs) *api.BillingInvoiceLineExternalIds {
	if e.Invoicing == "" {
		return nil
	}

	return &api.BillingInvoiceLineExternalIds{
		Invoicing: lo.ToPtr(e.Invoicing),
	}
}
