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

package commonhttp

import (
	"encoding/json"
)

type Union[Primary any, Secondary any] struct {
	Option1 *Primary
	Option2 *Secondary
}

// Implements json.Marshaler with primary having precedence.
func (u Union[Primary, Secondary]) MarshalJSON() ([]byte, error) {
	if u.Option1 != nil {
		return json.Marshal(u.Option1)
	}
	if u.Option2 != nil {
		return json.Marshal(u.Option2)
	}
	// if nothing is set we return empty
	return []byte{}, nil
}
