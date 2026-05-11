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

	StatusRealizationStarted              Status = "active.realization.started"
	StatusRealizationWaitingForCollection Status = "active.realization.waiting_for_collection"
	StatusRealizationProcessing           Status = "active.realization.processing"
	StatusRealizationIssuing              Status = "active.realization.issuing"
	StatusRealizationCompleted            Status = "active.realization.completed"
	StatusAwaitingPaymentSettlement       Status = "active.awaiting_payment_settlement"

	StatusFinal   Status = Status(meta.ChargeStatusFinal)
	StatusDeleted Status = Status(meta.ChargeStatusDeleted)
)

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
		string(StatusRealizationStarted),
		string(StatusRealizationWaitingForCollection),
		string(StatusRealizationProcessing),
		string(StatusRealizationIssuing),
		string(StatusRealizationCompleted),
		string(StatusAwaitingPaymentSettlement),
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
