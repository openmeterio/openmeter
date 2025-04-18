package fixtures

import "github.com/oklog/ulid/v2"

func RandKey() string {
	return ulid.Make().String()
}

func RandULID() string {
	return ulid.Make().String()
}
