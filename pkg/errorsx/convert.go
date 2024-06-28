package errorsx

import "errors"

func ErrorAs[T error](err error) (T, bool) {
	var outerr T
	if errors.As(err, &outerr) {
		return outerr, true
	}
	return outerr, false
}
