// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
