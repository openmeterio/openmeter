package flatfee

import (
	"fmt"
	"slices"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Status string

const (
	StatusCreated Status = Status(meta.ChargeStatusCreated)
	StatusActive  Status = Status(meta.ChargeStatusActive)
	StatusFinal   Status = Status(meta.ChargeStatusFinal)
)

func (Status) Values() []string {
	return []string{
		string(StatusCreated),
		string(StatusActive),
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
	if err := s.Validate(); err != nil {
		return meta.ChargeStatusCreated, err
	}

	return meta.ChargeStatus(s), nil
}

type Trigger = stateless.Trigger

var TriggerNext Trigger = "trigger_next"
