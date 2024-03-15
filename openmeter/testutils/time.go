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

package testutils

import (
	"testing"
	"time"
)

func GetRFC3339Time(t *testing.T, timeString string) time.Time {
	t.Helper()
	t1, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}
	return t1
}
