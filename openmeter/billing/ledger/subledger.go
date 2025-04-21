package ledger

import (
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Subledger struct {
	models.ManagedResource

	Key      string
	LedgerID string
	Priority int64
}

func (s *Subledger) Validate() error {
	var errs []error

	if err := s.ManagedResource.Validate(); err != nil {
		errs = append(errs, err)
	}

	if s.LedgerID == "" {
		errs = append(errs, errors.New("ledger id is required"))
	}

	if s.Priority < 0 {
		errs = append(errs, errors.New("priority must be greater than 0"))
	}

	return errors.Join(errs...)
}

type UpsertSubledgerInput struct {
	Key         string
	Priority    int64
	Name        string
	Description *string
}

func (i UpsertSubledgerInput) Validate() error {
	var errs []error

	if i.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if i.Priority < 0 {
		errs = append(errs, errors.New("priority must be greater than 0"))
	}

	return errors.Join(errs...)
}

type UpsertSubledgerAdapterInput struct {
	UpsertSubledgerInput
	LedgerID LedgerID
}

func (i UpsertSubledgerAdapterInput) Validate() error {
	var errs []error

	if err := i.UpsertSubledgerInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.LedgerID.Validate(); err != nil {
		errs = append(errs, errors.New("ledger id is required"))
	}

	return errors.Join(errs...)
}
