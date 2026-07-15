package billinginvoices

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	apilegacy "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/addons"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	chargeshandler "github.com/openmeterio/openmeter/api/v3/handlers/customers/charges"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/labels"
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
func ToAPIBillingInvoice(inv billing.Invoice) (api.BillingInvoice, error) {
	switch inv.Type() {
	case billing.InvoiceTypeStandard:
		std, err := inv.AsStandardInvoice()
		if err != nil {
			return api.BillingInvoice{}, fmt.Errorf("reading standard invoice: %w", err)
		}

		return ToAPIStandardInvoice(std)
	default:
		genericInv, _ := inv.AsGenericInvoice()

		return api.BillingInvoice{}, billing.NotFoundError{
			ID:     genericInv.GetID(),
			Entity: billing.EntityInvoice,
			Err:    fmt.Errorf("unsupported invoice type %q", inv.Type()),
		}
	}
}

func ToAPIStandardInvoice(std billing.StandardInvoice) (api.BillingInvoice, error) {
	var out api.BillingInvoice
	stdAPI, err := toAPIStandardInvoice(std)
	if err != nil {
		return out, err
	}

	if err := out.FromBillingInvoiceStandard(stdAPI); err != nil {
		return out, fmt.Errorf("setting standard invoice union: %w", err)
	}

	return out, nil
}

func toAPIStandardInvoice(std billing.StandardInvoice) (api.BillingInvoiceStandard, error) {
	// Sort lines for consistent output — matches v1 behavior.
	std.SortLines()

	workflow, err := toAPIWorkflow(std.Workflow)
	if err != nil {
		return api.BillingInvoiceStandard{}, fmt.Errorf("converting workflow: %w", err)
	}

	outLines, err := mapLines(std.Lines.OrEmpty())
	if err != nil {
		return api.BillingInvoiceStandard{}, err
	}

	return api.BillingInvoiceStandard{
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
		Type:                  api.BillingInvoiceStandardTypeStandard,
		Status:                api.BillingInvoiceStandardStatus(std.Status.ShortStatus()),
		StatusDetails:         toAPIStatusDetails(std.StatusDetails, std.Status),
		Customer:              toAPIInvoiceCustomer(std.Customer),
		Supplier:              billingprofiles.ToAPIBillingSupplier(std.Supplier),
		Totals:                chargeshandler.ToAPIBillingTotals(std.Totals),
		ValidationIssues:      mapValidationIssues(std.ValidationIssues),
		ExternalReferences:    toAPIInvoiceExternalReferences(std.ExternalIDs),
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
	}

	if c.UsageAttribution != nil {
		out.UsageAttribution = &api.BillingCustomerUsageAttribution{
			SubjectKeys: c.UsageAttribution.SubjectKeys,
		}
	}

	if c.BillingAddress != nil && !lo.IsEmpty(*c.BillingAddress) {
		var country *api.CountryCode
		if c.BillingAddress.Country != nil {
			country = lo.ToPtr(api.CountryCode(*c.BillingAddress.Country))
		}
		out.BillingAddress = &api.Address{
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
		DueAfter:    lo.EmptyableToPtr(config.Invoicing.DueAfter.String()),
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
		Apps: &api.BillingInvoiceWorkflowAppsReferences{
			Invoicing: api.BillingAppReference{
				Id: w.AppReferences.Invoicing.ID,
			},
			Payment: api.BillingAppReference{
				Id: w.AppReferences.Payment.ID,
			},
			Tax: api.BillingAppReference{
				Id: w.AppReferences.Tax.ID,
			},
		},
		SourceBillingProfile: api.BillingProfileReference{
			Id: w.SourceBillingProfileID,
		},
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

func toAPIInvoiceExternalReferences(e externalid.InvoiceExternalIDs) *api.BillingInvoiceExternalReferences {
	if e.Invoicing == "" && e.Payment == "" {
		return nil
	}

	return &api.BillingInvoiceExternalReferences{
		InvoicingId: lo.EmptyableToPtr(e.Invoicing),
		PaymentId:   lo.EmptyableToPtr(e.Payment),
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
		if err := invoiceLine.FromBillingInvoiceStandardLine(mapped); err != nil {
			return api.BillingInvoiceLine{}, fmt.Errorf("setting standard line union: %w", err)
		}

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
		Id:                  lo.ToPtr(line.ID),
		Name:                line.Name,
		Description:         line.Description,
		Labels:              labels.FromMetadata(line.Metadata),
		CreatedAt:           line.CreatedAt,
		UpdatedAt:           line.UpdatedAt,
		DeletedAt:           line.DeletedAt,
		Type:                api.BillingInvoiceStandardLineTypeStandardLine,
		LifecycleController: chargeshandler.ConvertLifecycleControllerToAPI(line.ManagedBy),
		ServicePeriod:       chargeshandler.ConvertClosedPeriodToAPI(line.Period),
		Totals:              chargeshandler.ToAPIBillingTotals(line.Totals),
		Charge:              chargeRef,
		Subscription:        subRef,
		ExternalReferences:  toAPILineExternalReferences(line.ExternalIDs),
		CreditsApplied:      mapCreditApplies(line.CreditsApplied),
		Discounts:           mapLineDiscounts(line.Discounts),
		RateCard:            rateCard,
		DetailedLines:       detailedLines,
	}, nil
}

func mapRateCard(line *billing.StandardLine) (api.BillingInvoiceLineRateCard, error) {
	if line.UsageBased == nil {
		return api.BillingInvoiceLineRateCard{}, fmt.Errorf("standard line %s has no usage-based configuration", line.ID)
	}

	if line.UsageBased.Price == nil {
		return api.BillingInvoiceLineRateCard{}, fmt.Errorf("standard line %s has no price set", line.ID)
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
		Id:                 dl.ID,
		Name:               dl.Name,
		Description:        dl.Description,
		CreatedAt:          dl.CreatedAt,
		UpdatedAt:          dl.UpdatedAt,
		DeletedAt:          dl.DeletedAt,
		Category:           api.BillingInvoiceDetailedLineCostCategory(dl.Category),
		ServicePeriod:      chargeshandler.ConvertClosedPeriodToAPI(dl.ServicePeriod),
		Quantity:           dl.Quantity.String(),
		UnitPrice:          dl.PerUnitAmount.String(),
		Totals:             chargeshandler.ToAPIBillingTotals(dl.Totals),
		CreditsApplied:     mapCreditApplies(dl.CreditsApplied),
		Discounts:          mapAmountDiscounts(dl.AmountDiscounts),
		ExternalReferences: toAPILineExternalReferences(dl.ExternalIDs),
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
			Id:                 ud.ID,
			Quantity:           ud.Quantity.String(),
			Description:        ud.Description,
			ExternalReferences: toAPILineExternalReferences(ud.ExternalIDs),
			Reason:             api.BillingInvoiceDiscountReason(ud.Reason.Type()),
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
			Id:                 ad.ID,
			Amount:             ad.Amount.String(),
			Description:        ad.Description,
			ExternalReferences: toAPILineExternalReferences(ad.ExternalIDs),
			Reason:             api.BillingInvoiceDiscountReason(ad.Reason.Type()),
		}
	})

	return &api.BillingInvoiceLineDiscounts{
		Amount: &amountDiscounts,
	}
}

func toAPILineExternalReferences(e externalid.LineExternalIDs) *api.BillingInvoiceLineExternalReferences {
	if e.Invoicing == "" {
		return nil
	}

	return &api.BillingInvoiceLineExternalReferences{
		InvoicingId: lo.ToPtr(e.Invoicing),
	}
}

// invoiceSortField maps v3 sort field names to the billing domain's InvoiceOrderBy values.
func FromAPIInvoiceSortField(ctx context.Context, field string) (apilegacy.InvoiceOrderBy, error) {
	switch field {
	case "issued_at":
		return apilegacy.InvoiceOrderByIssuedAt, nil
	case "created_at":
		return apilegacy.InvoiceOrderByCreatedAt, nil
	case "service_period_start":
		return apilegacy.InvoiceOrderByPeriodStart, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "issued_at", "created_at", "service_period_start")
	}
}

// mergeStandardInvoiceFromAPI applies the mutable fields of an update request onto an
// already-loaded standard invoice, in place. Only description, labels, supplier, customer,
// workflow, and top-level lines are mutable; all other invoice fields are left untouched.
func mergeStandardInvoiceFromAPI(inv *billing.StandardInvoice, req api.UpdateInvoiceStandardRequest) error {
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
func mergeInvoiceSupplierFromAPI(existing billing.SupplierContact, updated api.UpdateSupplier) billing.SupplierContact {
	existing.Name = lo.FromPtrOr(updated.Name, "")

	if updated.Addresses != nil {
		existing.Address = billingprofiles.FromAPIAddress(api.Address(updated.Addresses.BillingAddress))
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
func mergeInvoiceCustomerFromAPI(existing billing.InvoiceCustomer, updated api.UpdateInvoiceCustomer) billing.InvoiceCustomer {
	existing.Name = updated.Name

	if updated.BillingAddress != nil {
		addr := billingprofiles.FromAPIAddress(api.Address(*updated.BillingAddress))
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
func mergeInvoiceWorkflowFromAPI(existing billing.InvoiceWorkflow, updated api.UpdateInvoiceWorkflowSettings) (billing.InvoiceWorkflow, error) {
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
func mergeStandardInvoiceLinesFromAPI(inv *billing.StandardInvoice, lines *[]api.UpdateInvoiceLine) (billing.StandardInvoiceLines, error) {
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
func standardLineFromAPI(line api.UpdateInvoiceStandardLine, inv *billing.StandardInvoice) (*billing.StandardLine, error) {
	price, taxConfig, featureKey, discounts, err := mapRateCardFromAPI(line.RateCard)
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
			RateCardDiscounts: discounts,
		},
		UsageBased: &billing.UsageBasedLine{
			Price:      price,
			FeatureKey: featureKey,
		},
	}, nil
}

// mergeStandardLineFromAPI applies the editable fields of an update request line onto an
// existing top-level standard line, matched by ID.
func mergeStandardLineFromAPI(existing *billing.StandardLine, line api.UpdateInvoiceStandardLine) (*billing.StandardLine, error) {
	price, taxConfig, featureKey, discounts, err := mapRateCardFromAPI(line.RateCard)
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
	existing.RateCardDiscounts = discounts
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
func mapRateCardFromAPI(rc api.UpdateInvoiceLineRateCard) (*productcatalog.Price, *billing.TaxConfig, string, billing.Discounts, error) {
	price, err := plans.FromAPIBillingPrice(api.BillingPrice(rc.Price), nil)
	if err != nil {
		return nil, nil, "", billing.Discounts{}, fmt.Errorf("mapping price: %w", err)
	}

	var discounts billing.Discounts
	if rc.Discounts != nil {
		pcDiscounts, err := plans.FromAPIBillingRateCardDiscounts(api.BillingRateCardDiscounts(*rc.Discounts))
		if err != nil {
			return nil, nil, "", billing.Discounts{}, fmt.Errorf("mapping discounts: %w", err)
		}

		discounts = billing.DiscountsFromProductCatalog(pcDiscounts).UpsertCorrelationIDs()
	}

	taxConfig := billing.FromProductCatalog(addons.FromAPIBillingRateCardTaxConfig(fromAPIUpdateRateCardTaxConfig(rc.TaxConfig)))

	return price, taxConfig, lo.FromPtrOr(rc.FeatureKey, ""), discounts, nil
}

func fromAPIUpdateRateCardTaxConfig(taxConfig *api.UpdateRateCardTaxConfig) *api.BillingRateCardTaxConfig {
	if taxConfig == nil {
		return nil
	}
	return &api.BillingRateCardTaxConfig{
		Behavior: taxConfig.Behavior,
		Code: api.TaxCodeReference{
			Id: taxConfig.Code.Id,
		},
	}
}
