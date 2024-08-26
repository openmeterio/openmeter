package models

import (
	"errors"
	"maps"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/convert"
)

type SubjectKeyAndID struct {
	Key string `json:"key"`
	ID  string `json:"id,omitempty"`
}

func (s SubjectKeyAndID) Validate() error {
	if s.Key == "" {
		return errors.New("key is required")
	}

	return nil
}

type Subject struct {
	Id                 *string                `json:"id"`
	Key                string                 `json:"key"`
	DisplayName        *string                `json:"displayName,omitempty"`
	Metadata           map[string]interface{} `json:"metadata"`
	CurrentPeriodStart *time.Time             `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time             `json:"currentPeriodEnd,omitempty"`
	StripeCustomerId   *string                `json:"stripeCustomerId,omitempty"`
}

func (s Subject) Validate() error {
	if s.Key == "" {
		return errors.New("key is required")
	}

	return nil
}

func (s Subject) ToAPIModel() api.Subject {
	return api.Subject{
		Id:                 s.Id,
		Key:                s.Key,
		DisplayName:        s.DisplayName,
		Metadata:           convert.ToPointer(maps.Clone(s.Metadata)),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		StripeCustomerId:   s.StripeCustomerId,
	}
}
