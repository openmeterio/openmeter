package subject

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Subject represents a subject in the system.
type Subject struct {
	models.ManagedModel

	Namespace   string                 `json:"namespace"`
	Id          string                 `json:"id"`
	Key         string                 `json:"key"`
	DisplayName *string                `json:"displayName,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	// Use customer application entity instead
	StripeCustomerId *string `json:"stripeCustomerId,omitempty"`
}

func (s Subject) Validate() error {
	var errs []error

	if s.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	return errors.Join(errs...)
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
