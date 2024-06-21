package entutils

import (
	"database/sql/driver"

	"github.com/oklog/ulid/v2"
)

// ULID implements a valuer (and Scanner) that can serialize string ULIDs into postgres
// instead of the binary representation, as postgres interprets those as UTF-8 strings
type ULID struct {
	ulid.ULID
}

func (v ULID) Value() (driver.Value, error) {
	return v.ULID.String(), nil
}

func (v *ULID) ULIDPointer() *ulid.ULID {
	if v == nil {
		return nil
	}
	return &v.ULID
}

func Ptr(u *ulid.ULID) *ULID {
	if u == nil {
		return nil
	}

	return &ULID{
		ULID: *u,
	}
}

func Wrap(u ulid.ULID) ULID {
	return ULID{
		ULID: u,
	}
}
