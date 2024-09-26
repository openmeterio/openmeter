package app

import (
	"fmt"
)

var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
}

var _ error = (*MarketplaceListingNotFoundError)(nil)

type MarketplaceListingNotFoundError struct {
	MarketplaceListingID
}

func (e MarketplaceListingNotFoundError) Error() string {
	return fmt.Sprintf("listing with key %s not found", e.Key)
}

type genericError struct {
	Err error
}

var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}
