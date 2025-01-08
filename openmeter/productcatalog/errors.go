package productcatalog

var _ error = (*ValidationError)(nil)

type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func NewValidationError(err error) error {
	if err != nil {
		return &ValidationError{err}
	}

	return nil
}
