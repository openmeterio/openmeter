package flatfee

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Status string

const (
	StatusCreated Status = Status(meta.ChargeStatusCreated)
	StatusActive  Status = Status(meta.ChargeStatusActive)

	StatusActiveRealizationStarted              Status = "active.realization.started"
	StatusActiveRealizationWaitingForCollection Status = "active.realization.waiting_for_collection"
	StatusActiveRealizationProcessing           Status = "active.realization.processing"
	StatusActiveRealizationIssuing              Status = "active.realization.issuing"
	StatusActiveRealizationCompleted            Status = "active.realization.completed"
	StatusActiveAwaitingPaymentSettlement       Status = "active.awaiting_payment_settlement"

	StatusFinal   Status = Status(meta.ChargeStatusFinal)
	StatusDeleted Status = Status(meta.ChargeStatusDeleted)
)

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
		string(StatusActiveRealizationStarted),
		string(StatusActiveRealizationWaitingForCollection),
		string(StatusActiveRealizationProcessing),
		string(StatusActiveRealizationIssuing),
		string(StatusActiveRealizationCompleted),
		string(StatusActiveAwaitingPaymentSettlement),
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

	split := strings.SplitN(string(s), ".", 2)
	if len(split) == 0 {
		return meta.ChargeStatusCreated, fmt.Errorf("invalid status: %s", s)
	}

	metaStatus := meta.ChargeStatus(split[0])
	if err := metaStatus.Validate(); err != nil {
		return meta.ChargeStatusCreated, fmt.Errorf("invalid status: %s", s)
	}

	return metaStatus, nil
}
