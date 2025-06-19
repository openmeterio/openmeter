package subscription

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// NewSubscriptionNotFoundError returns a new SubscriptionNotFoundError.
func NewSubscriptionNotFoundError(id string) error {
	return &SubscriptionNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription %s not found", id),
		),
	}
}

var _ models.GenericError = &SubscriptionNotFoundError{}

// SubscriptionNotFoundError is returned when a meter is not found.
type SubscriptionNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *SubscriptionNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *SubscriptionNotFoundError) Unwrap() error {
	return e.err
}

// IsSubscriptionNotFoundError returns true if the error is a SubscriptionNotFoundError.
func IsSubscriptionNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *SubscriptionNotFoundError

	return errors.As(err, &e)
}

// NewPhaseNotFoundError returns a new PhaseNotFoundError.
func NewPhaseNotFoundError(phaseId string) error {
	return &PhaseNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription phase %s not found", phaseId),
		),
	}
}

var _ models.GenericError = &PhaseNotFoundError{}

// PhaseNotFoundError is returned when a meter is not found.
type PhaseNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *PhaseNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *PhaseNotFoundError) Unwrap() error {
	return e.err
}

// IsPhaseNotFoundError returns true if the error is a PhaseNotFoundError.
func IsPhaseNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *PhaseNotFoundError

	return errors.As(err, &e)
}

// NewItemNotFoundError returns a new ItemNotFoundError.
func NewItemNotFoundError(itemId string) error {
	return &ItemNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription item %s not found", itemId),
		),
	}
}

var _ models.GenericError = &ItemNotFoundError{}

// ItemNotFoundError is returned when a meter is not found.
type ItemNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *ItemNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *ItemNotFoundError) Unwrap() error {
	return e.err
}

// IsItemNotFoundError returns true if the error is a ItemNotFoundError.
func IsItemNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *ItemNotFoundError

	return errors.As(err, &e)
}

type BillingPeriodQueriedBeforeSubscriptionStartError struct {
	err error
}

func (e *BillingPeriodQueriedBeforeSubscriptionStartError) Error() string {
	return e.err.Error()
}

func (e *BillingPeriodQueriedBeforeSubscriptionStartError) Unwrap() error {
	return e.err
}

func NewBillingPeriodQueriedBeforeSubscriptionStartError(queriedAt, subscriptionStart time.Time) error {
	return &BillingPeriodQueriedBeforeSubscriptionStartError{
		err: models.NewGenericValidationError(
			fmt.Errorf("billing period queried before subscription start: %s < %s", queriedAt, subscriptionStart),
		),
	}
}

func IsBillingPeriodQueriedBeforeSubscriptionStartError(err error) bool {
	if err == nil {
		return false
	}

	var e *BillingPeriodQueriedBeforeSubscriptionStartError

	return errors.As(err, &e)
}
