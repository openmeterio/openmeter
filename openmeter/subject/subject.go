package subject

import (
	"errors"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Subject represents a subject in the system.
type Subject struct {
	Id          string
	Key         string
	DisplayName *string
	Metadata    models.Metadata
	// Use customer application entity instead
	StripeCustomerId *string
}

func (s Subject) Validate() error {
	var errs []error

	if s.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	return errors.Join(errs...)
}

// ToAPIModel converts the subject to the API model.
func (s Subject) ToAPIModel() api.Subject {
	var metadata *map[string]string

	if s.Metadata != nil {
		m := map[string]string{}

		for k, v := range s.Metadata {
			m[k] = v
		}

		metadata = &m
	}

	return api.Subject{
		Id:               s.Id,
		Key:              s.Key,
		DisplayName:      s.DisplayName,
		Metadata:         metadata,
		StripeCustomerId: s.StripeCustomerId,
	}
}

// SubjectKey is key only version of Subject
// Used in in entitlements events to reduce payload size
type SubjectKey struct {
	Key string `json:"key"`
}

func (s SubjectKey) Validate() error {
	if s.Key == "" {
		return errors.New("key is required")
	}

	return nil
}
