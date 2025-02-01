package entity

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ProgressID is the identifier for a progress
type ProgressID struct {
	models.NamespacedModel `json:"namespace,inline"`
	ID                     string
}

func (a *ProgressID) Validate() error {
	var errs []error

	if err := a.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced model: %w", err))
	}

	if a.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return errors.Join(errs...)
}

// Progress is the identifier for a progress
type Progress struct {
	ProgressID `json:"id"`

	// Success is the number of items that succeeded
	Success uint64 `json:"success"`
	// Failed is the number of items that failed
	Failed uint64 `json:"failed"`
	// The total number of items to process
	Total uint64 `json:"total"`
	// The time the progress was last updated
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a *Progress) Validate() error {
	var errs []error

	if err := a.ProgressID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("progress id: %w", err))
	}

	if a.Success+a.Failed > a.Total {
		errs = append(errs, errors.New("success and failed must be less than or equal to total"))
	}

	if a.UpdatedAt.IsZero() {
		errs = append(errs, errors.New("updated at is required"))
	}

	return errors.Join(errs...)
}

// Get progress is the input for the GetProgress method
type GetProgressInput struct {
	ProgressID
}

func (a *GetProgressInput) Validate() error {
	var errs []error

	if err := a.ProgressID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("progress id: %w", err))
	}

	return errors.Join(errs...)
}

// UpsertProgressInput is the input for the UpsertProgress method
type UpsertProgressInput struct {
	Progress
}

func (a *UpsertProgressInput) Validate() error {
	var errs []error

	if err := a.Progress.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("progress: %w", err))
	}

	return errors.Join(errs...)
}
