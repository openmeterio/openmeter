package ledger

import "errors"

type OwnerReferenceType string

type OwnerReference struct {
	Type OwnerReferenceType
	ID   string
}

func (o OwnerReference) Validate() error {
	var errs []error

	if o.Type == "" {
		errs = append(errs, errors.New("type is required"))
	}

	if o.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return errors.Join(errs...)
}
