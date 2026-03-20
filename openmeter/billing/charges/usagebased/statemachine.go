package usagebased

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Status string

const (
	StatusCreated Status = Status(meta.ChargeStatusCreated)

	// Active status and substates
	StatusActive Status = Status(meta.ChargeStatusActive)

	StatusActiveFinalRealizationStarted              Status = "active.final_realization.started"
	StatusActiveFinalRealizationWaitingForCollection Status = "active.final_realization.waiting_for_collection"
	StatusActiveFinalRealizationProcessing           Status = "active.final_realization.processing"
	StatusActiveFinalRealizationCompleted            Status = "active.final_realization.completed"

	StatusFinal Status = Status(meta.ChargeStatusFinal)
)

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
		string(StatusActiveFinalRealizationStarted),
		string(StatusActiveFinalRealizationWaitingForCollection),
		string(StatusActiveFinalRealizationProcessing),
		string(StatusActiveFinalRealizationCompleted),
		string(StatusFinal),
	}
}

func (s Status) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid status: %s", s)
	}
	return nil
}

func (s Status) ToMetaChargeStatus() (meta.ChargeStatus, error) {
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
