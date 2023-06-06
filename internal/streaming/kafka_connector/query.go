package kafka_connector

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/openmeterio/openmeter/internal/models"
	. "github.com/openmeterio/openmeter/internal/streaming"
)

// https://github.com/cloudevents/spec/blob/main/cloudevents/formats/cloudevents.json
//
//go:embed sql/events_stream.tpl.sql
var cloudEventsStreamQueryTemplate string

type cloudEventsStreamQueryData struct {
	Topic      string
	Partitions int
}

//go:embed sql/detected_events_table.tpl.sql
var detectedEventsTableQueryTemplate string

type detectedEventsTableQueryData struct {
	Retention  int
	Partitions int
}

//go:embed sql/detected_events_stream.tpl.sql
var detectedEventsStreamQueryTemplate string

type detectedEventsStreamQueryData struct{}

//go:embed sql/meter_table_describe.tpl.sql
var meterTableDescribeQueryTemplate string

//go:embed sql/values.tpl.sql
var meterValuesTemplate string

type meterValuesData struct {
	*models.Meter
	*GetValuesParams
}

type meterTableDescribeQueryData struct {
	*models.Meter
}

//go:embed sql/meter_table.tpl.sql
var meterTableQueryTemplate string

type meterTableQueryData struct {
	*models.Meter
	WindowSize      string
	WindowRetention string
	Partitions      int
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

func Execute(temp string, data any) (string, error) {
	tmpl := template.New("sql")
	tmpl.Funcs(funcMap())

	t, err := tmpl.Parse(temp)
	if err != nil {
		return "", err
	}

	b := bytes.NewBufferString("")
	err = t.Execute(b, data)
	if err != nil {
		return "", err
	}

	return sanitizeQuery(b.String()), nil
}

func sanitizeQuery(content string) string {
	r := regexp.MustCompile(`\s+`)
	content = r.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)
	return content
}
