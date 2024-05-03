package api

import "errors"

var (
	ErrCreditQueryLimitOutOfRange = errors.New("query limit is out of range")
)

func ValidateCreditQueryLimit(l *CreditQueryLimit) (CreditQueryLimit, error) {
	if l == nil {
		return DefaultCreditQueryLimit, nil
	}

	limit := *l

	if limit <= 0 || limit > MaxCreditQueryLimit {
		return 0, ErrCreditQueryLimitOutOfRange
	}

	return limit, nil
}
