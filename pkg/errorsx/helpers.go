package errorsx

import (
	"errors"
	"reflect"
	"slices"

	api "github.com/openmeterio/openmeter/api"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

var apiErrorPackages = []string{
	reflect.TypeOf(api.InvalidParamFormatError{}).PkgPath(),
	reflect.TypeOf(apiv3.InvalidParamFormatError{}).PkgPath(),
}

func isAPIError(err error) bool {
	// Package matching is the most maintainable way to identify generated API errors.
	// The codegen output does not provide a common base error or helper for them.
	return isErrorFromPackages(err, apiErrorPackages)
}

func isErrorFromPackages(err error, packagePaths []string) bool {
	if err == nil {
		return false
	}

	t := reflect.TypeOf(err)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if slices.Contains(packagePaths, t.PkgPath()) {
		return true
	}

	return isUnwrappedErrorFromPackages(err, packagePaths)
}

func isUnwrappedErrorFromPackages(err error, packagePaths []string) bool {
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		return isErrorFromPackages(unwrapped, packagePaths)
	}

	if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		for _, err := range unwrapper.Unwrap() {
			if isErrorFromPackages(err, packagePaths) {
				return true
			}
		}
	}

	return false
}
