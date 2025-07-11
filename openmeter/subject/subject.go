package subject

import (
	"errors"
)

// Subject represents a subject in the system.
type Subject struct {
	Namespace   string
	Id          string
	Key         string
	DisplayName *string
	Metadata    map[string]interface{}
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
