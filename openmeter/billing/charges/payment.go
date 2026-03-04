package charges

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	PaymentSettlementStatusAttributeKey = "payment_settlement_status"
	PaymentSettlementTypeAttributeKey   = "payment_settlement_type"

	PaymentSettlementTypeExternal        = CreditPurchaseSettlementTypeExternal
	PaymentSettlementTypeStandardInvoice = CreditPurchaseSettlementTypeInvoice
)

type PaymentSettlementStatus string

const (
	PaymentSettlementStatusAuthorized PaymentSettlementStatus = "authorized"
	PaymentSettlementStatusSettled    PaymentSettlementStatus = "settled"
)

func (o PaymentSettlementStatus) Values() []string {
	return []string{
		string(PaymentSettlementStatusAuthorized),
		string(PaymentSettlementStatusSettled),
	}
}

func (o PaymentSettlementStatus) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid standard invoice payment settlement status: %s", o)
	}
	return nil
}

// PaymentSettlementBase the generic payment settlement properties that are common to all payment settlements.
type PaymentSettlementBase struct {
	models.NamespacedID
	models.ManagedModel

	Annotations   models.Annotations    `json:"annotations"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	Status PaymentSettlementStatus `json:"status"`
	Amount alpacadecimal.Decimal   `json:"amount"`

	Authorized *TimedLedgerTransactionGroupReference `json:"authorized"`
	Settled    *TimedLedgerTransactionGroupReference `json:"settled"`
}

func (r PaymentSettlementBase) Validate() error {
	var errs []error

	if err := r.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := r.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if r.Authorized != nil {
		if err := r.Authorized.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("authorized: %w", err))
		}
	}

	if r.Settled != nil {
		if err := r.Settled.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("settled: %w", err))
		}
	}

	if !r.Amount.IsPositive() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	switch r.Status {
	case PaymentSettlementStatusAuthorized:
		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorization transaction data is missing for authorized status"))
		}
	case PaymentSettlementStatusSettled:
		if r.Settled == nil {
			errs = append(errs, fmt.Errorf("settlement transaction data is missing for settled status"))
		}

		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorization transaction data is missing for settled status"))
		}
	}

	return errors.Join(errs...)
}

var _ models.Validator = (*ExternalPaymentSettlement)(nil)

type ExternalPaymentSettlement PaymentSettlementBase

func (r ExternalPaymentSettlement) Validate() error {
	return PaymentSettlementBase(r).Validate()
}

func (r ExternalPaymentSettlement) ErrorAttributes() models.Attributes {
	return models.Attributes{
		PaymentSettlementStatusAttributeKey: string(r.Status),
		PaymentSettlementTypeAttributeKey:   string(PaymentSettlementTypeExternal),
	}
}
