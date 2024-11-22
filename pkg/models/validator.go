package models

type Validator interface {
	// Validate returns an error if the instance of the Validator is invalid.
	Validate() error
}
