package events

import (
	_ "embed"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed events-schema.gen.json
var eventsSchema string

type SchemaFileLoader struct {
	jsonschema.FileLoader
}

func (l SchemaFileLoader) Load(url string) (any, error) {
	return jsonschema.UnmarshalJSON(strings.NewReader(eventsSchema))
}

func SchemaValidator() (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()

	c.UseLoader(SchemaFileLoader{jsonschema.FileLoader{}})

	return c.Compile("whatever")
}
