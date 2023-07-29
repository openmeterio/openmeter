package clickhouse_connector

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

//go:embed sql/create_events_table.tpl.sql
var createEventsTableTemplate string

type createEventsTableData struct {
	Database        string
	EventsTableName string
}

//go:embed sql/create_meter_view.tpl.sql
var createMeterViewTemplate string

type createMeterViewData struct {
	Database        string
	MeterViewName   string
	MeterSlug       string
	EventsTableName string
	GroupBy         map[string]string
}

// TODO: consolidate between ksql's query.go file and this one
func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")

	return f
}

func templateQuery(temp string, data any) (string, error) {
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
