package streaming

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
)

func TemplateQuery(temp string, data any) (string, error) {
	tmpl := template.New("sql")
	tmpl.Funcs(funcMap())

	t, err := tmpl.Parse(temp)
	if err != nil {
		return "", fmt.Errorf("parse query: %w", err)
	}

	b := bytes.NewBufferString("")
	err = t.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("template query: %w", err)
	}

	return sanitizeQuery(b.String()), nil
}

func sanitizeQuery(content string) string {
	r := regexp.MustCompile(`\s+`)
	content = r.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)
	return content
}

func deref[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}

func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")

	f["bquote"] = func(str ...interface{}) string {
		out := make([]string, 0, len(str))
		for _, s := range str {
			if s != nil {
				out = append(out, fmt.Sprintf("`%v`", s))
			}
		}
		return strings.Join(out, " ")
	}

	f["derefstr"] = func(str *string) string {
		return deref(str)
	}

	f["dereftime"] = func(i *time.Time) time.Time {
		return deref(i)
	}

	f["unixEpochMs"] = func(date time.Time) string {
		return strconv.FormatInt(date.UnixMilli(), 10)
	}

	return f
}
