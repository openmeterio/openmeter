package webhook

var _ error = (*ValidationError)(nil)

type ValidationError struct {
	err error
}

func (e ValidationError) Error() string {
	return e.err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.err
}

func NewValidationError(err error) error {
	if err == nil {
		return nil
	}

	return ValidationError{err: err}
}
