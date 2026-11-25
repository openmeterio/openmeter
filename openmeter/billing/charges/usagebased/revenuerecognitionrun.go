package usagebased

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type RevenueRecognitionRunID models.NamespacedID

func (i RevenueRecognitionRunID) Validate() error {
	return models.NamespacedID(i).Validate()
}

// RevenueRecognitionRun records what changed on a charge since its previous
// revenue recognition run. It is independent of realization runs: it never has
// an invoice line, invoice, or payment, and a correction is a new run rather
// than an update of an old one.
type RevenueRecognitionRun struct {
	ID RevenueRecognitionRunID
	models.ManagedModel

	ChargeID  string `json:"chargeId"`
	FeatureID string `json:"featureId"`

	StoredAtLT time.Time `json:"storedAtLT"`
	// ServicePeriodTo is the end of the period the run covers.
	ServicePeriodTo time.Time `json:"servicePeriodTo"`
	// PriorRevRecogRunID identifies the preceding revenue recognition run. It is
	// nil for the first run.
	PriorRevRecogRunID *RevenueRecognitionRunID `json:"priorRevRecogRunId,omitempty"`
	// MeteredQuantity is the metered quantity for time IN [intent.servicePeriod.from, servicePeriodTo) capped by stored_at < StoredAtLT.
	MeteredQuantity alpacadecimal.Decimal `json:"meteredQuantity"`
	// Totals is the change since the previous revenue recognition run.
	Totals                                totals.DeltaTotals `json:"totals"`
	DetailedLinesIncludeCreditAllocations bool               `json:"detailedLinesIncludeCreditAllocations"`

	// CreditsAllocated and InvoicedUsages never carry CorrectsRealizationID: the
	// revenue recognition tables have no column for it.
	CreditsAllocated creditrealization.Realizations `json:"creditsAllocated"`
	// InvoicedUsages is the part of the run that credits do not cover. The run's
	// invoiced total is the sum of their amounts.
	InvoicedUsages creditrealization.Realizations   `json:"invoicedUsages"`
	DetailedLines  []RevenueRecognitionDetailedLine `json:"detailedLines"`
}

type RevenueRecognitionRuns []RevenueRecognitionRun

// RevenueRecognitionDetailedLine is a line of a revenue recognition run. It is
// narrower than a realization run's detailed line: it has one amount instead
// of totals, and no category, payment term, or invoicing app reference.
type RevenueRecognitionDetailedLine struct {
	models.ManagedResource

	Annotations models.Annotations `json:"annotations,omitempty"`
	Metadata    models.Metadata    `json:"metadata,omitempty"`

	Currency               currencyx.Code        `json:"currency"`
	ServicePeriod          timeutil.ClosedPeriod `json:"servicePeriod"`
	ChildUniqueReferenceID string                `json:"childUniqueReferenceID"`
	Index                  *int                  `json:"index,omitempty"`

	Quantity      alpacadecimal.Decimal `json:"quantity"`
	PerUnitAmount alpacadecimal.Decimal `json:"perUnitAmount"`
	// Amount is the line amount before discounts and credits. Like Quantity, it
	// is negative when the line corrects earlier usage.
	Amount alpacadecimal.Decimal `json:"amount"`

	AmountDiscounts chargedetailedline.AmountDiscounts `json:"amountDiscounts,omitempty"`
	CreditsApplied  creditsapplied.CreditsApplied      `json:"creditsApplied,omitempty"`
}

var _ models.Validator = (*RevenueRecognitionDetailedLine)(nil)

func (l RevenueRecognitionDetailedLine) Validate() error {
	var errs []error

	if err := l.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := l.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if l.ChildUniqueReferenceID == "" {
		errs = append(errs, errors.New("child unique reference id must be set"))
	}

	if l.PerUnitAmount.IsNegative() {
		errs = append(errs, errors.New("per unit amount is negative"))
	}

	if err := l.AmountDiscounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("amount discounts: %w", err))
	}

	if err := l.CreditsApplied.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credits applied: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateRevenueRecognitionRunInput struct {
	ChargeID                              meta.ChargeID                  `json:"chargeId"`
	FeatureID                             string                         `json:"featureId"`
	StoredAtLT                            time.Time                      `json:"storedAtLT"`
	ServicePeriodTo                       time.Time                      `json:"servicePeriodTo"`
	PriorRevRecogRunID                    *RevenueRecognitionRunID       `json:"priorRevRecogRunId,omitempty"`
	MeteredQuantity                       alpacadecimal.Decimal          `json:"meteredQuantity"`
	Totals                                totals.DeltaTotals             `json:"totals"`
	DetailedLinesIncludeCreditAllocations bool                           `json:"detailedLinesIncludeCreditAllocations"`
	CreditsAllocated                      creditrealization.CreateInputs `json:"creditsAllocated"`
	InvoicedUsages                        creditrealization.CreateInputs `json:"invoicedUsages"`
	// DetailedLines take their namespace from the charge.
	DetailedLines []RevenueRecognitionDetailedLine `json:"detailedLines"`
}

var _ models.Validator = (*CreateRevenueRecognitionRunInput)(nil)

func (i CreateRevenueRecognitionRunInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge id: %w", err))
	}

	if i.FeatureID == "" {
		errs = append(errs, errors.New("feature id must be set"))
	}

	if i.StoredAtLT.IsZero() {
		errs = append(errs, errors.New("stored at lt must be set"))
	}

	if i.ServicePeriodTo.IsZero() {
		errs = append(errs, errors.New("service period to must be set"))
	}

	if i.PriorRevRecogRunID != nil {
		if err := i.PriorRevRecogRunID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("prior revenue recognition run id: %w", err))
		}

		if i.PriorRevRecogRunID.Namespace != i.ChargeID.Namespace {
			errs = append(errs, errors.New("prior revenue recognition run namespace must match charge namespace"))
		}
	}

	if err := i.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if err := validateRevenueRecognitionRealizations(i.CreditsAllocated); err != nil {
		errs = append(errs, fmt.Errorf("credits allocated: %w", err))
	}

	if err := validateRevenueRecognitionRealizations(i.InvoicedUsages); err != nil {
		errs = append(errs, fmt.Errorf("invoiced usages: %w", err))
	}

	for idx, line := range i.DetailedLines {
		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("detailed lines[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// validateRevenueRecognitionRealizations replaces
// creditrealization.CreateInput.Validate for revenue recognition runs. The
// shared check requires a correction to name the realization it corrects, but
// these tables have no column for that, so setting it is rejected instead of
// dropped silently.
func validateRevenueRecognitionRealizations(inputs creditrealization.CreateInputs) error {
	var errs []error

	for idx, input := range inputs {
		if err := input.ServicePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: service period: %w", idx, err))
		}

		if err := input.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: type: %w", idx, err))
		}

		switch input.Type {
		case creditrealization.TypeAllocation:
			if !input.Amount.IsPositive() {
				errs = append(errs, fmt.Errorf("[%d]: amount must be positive", idx))
			}
		case creditrealization.TypeCorrection:
			if input.Amount.IsPositive() {
				errs = append(errs, fmt.Errorf("[%d]: amount must not be positive", idx))
			}
		}

		if err := input.LedgerTransaction.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: ledger transaction: %w", idx, err))
		}

		if input.LineID != nil && *input.LineID == "" {
			errs = append(errs, fmt.Errorf("[%d]: line id must be non-empty", idx))
		}

		if input.CorrectsRealizationID != nil {
			errs = append(errs, fmt.Errorf("[%d]: corrects realization id is not stored for revenue recognition runs", idx))
		}
	}

	return errors.Join(errs...)
}
