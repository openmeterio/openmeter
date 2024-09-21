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

package config

import "strings"

// AddPrefix returns string with "<prefix>." prepended to key.
// If returns key unmodified if prefix is empty or key already has the prefix added.
func AddPrefix(prefix, key string) string {
	if prefix == "" || strings.HasPrefix(key, prefix+".") {
		return key
	}

	return prefix + "." + key
}
