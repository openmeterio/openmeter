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

package strcase

import (
	"strings"
	"unicode"
)

func SnakeToCamel(snake string) string {
	isToUpper := false
	camel := ""

	for i, ch := range snake {
		if ch == '_' {
			isToUpper = true
		} else {
			if isToUpper && i > 0 {
				camel += string(unicode.ToUpper(ch))
				isToUpper = false
			} else {
				camel += string(ch)
			}
		}
	}

	return camel
}

func CamelToSnake(camel string) string {
	var snake strings.Builder

	for i, ch := range camel {
		if unicode.IsUpper(ch) {
			if i > 0 {
				snake.WriteRune('_')
			}
			snake.WriteRune(unicode.ToLower(ch))
		} else {
			snake.WriteRune(ch)
		}
	}

	return snake.String()
}
