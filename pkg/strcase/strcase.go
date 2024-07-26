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
