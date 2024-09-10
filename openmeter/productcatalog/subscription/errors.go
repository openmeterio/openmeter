package subscription

type ForbiddenError struct{}

func (e *ForbiddenError) Error() string {
	panic("not implemented")
}
