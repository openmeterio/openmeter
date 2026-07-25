package creditpurchase

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Status string

const (
	StatusCreated                        Status = Status(meta.ChargeStatusCreated)
	StatusActive                         Status = Status(meta.ChargeStatusActive)
	StatusActivePaymentPending           Status = "active.payment.pending"
	StatusActiveInitialCreditGrant       Status = "active.initial_credit_grant"
	StatusActivePaymentAuthorized        Status = "active.payment.authorized"
	StatusActivePaymentSettled           Status = "active.payment.settled"
	StatusActivePaymentPaidAndAuthorized Status = "active.payment.paid_and_authorized"
	StatusFinal                          Status = Status(meta.ChargeStatusFinal)
	StatusDeleted                        Status = Status(meta.ChargeStatusDeleted)
)

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
		string(StatusActiveInitialCreditGrant),
		string(StatusActivePaymentPending),
		string(StatusActivePaymentAuthorized),
		string(StatusActivePaymentPaidAndAuthorized),
		string(StatusActivePaymentSettled),
		string(StatusFinal),
		string(StatusDeleted),
	}
}

func (s Status) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid status: %s", s))
	}
	return nil
}

func (s Status) ToMetaChargeStatus() (meta.ChargeStatus, error) {
	if err := s.Validate(); err != nil {
		return meta.ChargeStatusCreated, err
	}

	return meta.DetailedStatusToMetaStatus(string(s))
}
