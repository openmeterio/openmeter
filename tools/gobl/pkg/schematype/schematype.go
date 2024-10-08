package schematype

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	TypeObject        = "object"
	TypeInteger       = "integer"
	TypeString        = "string"
	TypeFloat64       = "float64"
	TypeBoolean       = "boolean"
	TypeGenericObject = "genericObject"
)

type TypeRef struct {
	IsObjectReference bool
	TargetObject      string
	TargetRef         *jsonschema.Schema

	Type     string
	Name     string
	Location string

	ArrayOf *TypeRef
}

func (p TypeRef) GetTargetRef() *jsonschema.Schema {
	if p.ArrayOf != nil {
		return p.ArrayOf.GetTargetRef()
	}

	return p.TargetRef
}

func (t TypeRef) String() (string, error) {
	if t.ArrayOf != nil {
		name, err := t.ArrayOf.String()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s[]", name), nil
	}

	name, err := locationToTypeName(t.TargetObject)
	if err != nil {
		return "", err
	}

	return name, nil
}

func locationToTypeName(location string) (string, error) {
	parsedUrl, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	// Hack remove draft-0 prefix
	components := strings.Split(parsedUrl.Path, "/")[2:]

	for _, fragment := range strings.Split(parsedUrl.Fragment, "/") {
		if fragment == "$defs" || fragment == "properties" {
			continue
		}

		components = append(components, fragment)
	}

	for i := range components {
		components[i] = strings.Title(components[i])
	}

	return strings.Join(components, ""), nil
}

func NewFromSchema(name string, prop *jsonschema.Schema) (*TypeRef, error) {
	ref, err := newFromSchema(name, prop)
	if err != nil {
		return nil, err
	}

	ref.Name, err = locationToTypeName(ref.GetTargetRef().Location)
	if err != nil {
		return nil, err
	}

	ref.Location = ref.GetTargetRef().Location

	return ref, nil
}

func newFromSchema(name string, prop *jsonschema.Schema) (*TypeRef, error) {
	if prop.Types == nil {
		// We are just holding a reference to another schema, let's try to resolve it

		for prop.Types == nil {
			prop = prop.Ref
		}
	}

	types := prop.Types.ToStrings()
	fmt.Printf("%s=>%v (%s)\n", name, types, prop.Location)

	if len(types) != 1 {
		panic("multiple types/zero")
	}

	// Let's handle the primitive types first
	t := types[0]
	entType := TypeRef{}

	switch t {
	case "string":
		if prop.Location == "" {
			// TODO: we need to just add a string field in this case

			panic("primitive type without location")
		} else {
			if prop.Ref == nil {
				// Final type
				entType = TypeRef{
					IsObjectReference: false,
					Type:              TypeString,
					TargetObject:      prop.Location,
					TargetRef:         prop,
				}
			} else {
				entType = TypeRef{
					IsObjectReference: true,
					TargetObject:      prop.Location,
					TargetRef:         prop,
					Type:              TypeObject,
				}
			}
		}
	case "integer":
		if prop.Location == "" {
			panic("primitive type without location")
		} else {
			if prop.Ref == nil {
				// Final type
				entType = TypeRef{
					IsObjectReference: false,
					Type:              TypeInteger,
					TargetObject:      prop.Location,
					TargetRef:         prop,
				}
			} else {
				entType = TypeRef{
					IsObjectReference: true,
					TargetObject:      prop.Location,
					TargetRef:         prop,
					Type:              TypeObject,
				}
			}
		}
	case "number":
		// float64 ??
		if prop.Location == "" {
			panic("primitive type without location")
		} else {
			if prop.Ref == nil {
				// Final type
				entType = TypeRef{
					IsObjectReference: false,
					Type:              TypeFloat64,
					TargetObject:      prop.Location,
					TargetRef:         prop,
				}
			} else {
				entType = TypeRef{
					IsObjectReference: true,
					TargetObject:      prop.Location,
					TargetRef:         prop,
					Type:              TypeObject,
				}
			}
		}
	case "boolean":
		// boolean ??
		if prop.Location == "" {
			panic("primitive type without location")
		} else {
			if prop.Ref == nil {
				// Final type
				entType = TypeRef{
					IsObjectReference: false,
					Type:              TypeBoolean,
					TargetObject:      prop.Location,
					TargetRef:         prop,
				}
			} else {
				entType = TypeRef{
					IsObjectReference: true,
					TargetObject:      prop.Location,
					TargetRef:         prop,
					Type:              TypeObject,
				}
			}
		}
	case "object":
		if prop.Location == "" {
			panic("object type without location => untyped object")
		}

		if len(prop.Properties) == 0 {
			// This is a final type, object without schema spec
			entType = TypeRef{
				IsObjectReference: true,
				TargetObject:      prop.Location,
				TargetRef:         prop,
				Type:              TypeGenericObject,
			}
		} else {
			entType = TypeRef{
				IsObjectReference: true,
				TargetObject:      prop.Location,
				TargetRef:         prop,
				Type:              TypeObject,
			}
		}

	case "array":
		if prop.Items2020 == nil {
			panic("array without items schema")
		}

		targetType, err := NewFromSchema(name, prop.Items2020)
		if err != nil {
			return nil, err
		}

		entType = TypeRef{
			TargetRef: prop,
			ArrayOf:   targetType,
		}
	default:
		panic("cannot handle ent type")
	}

	return &entType, nil
}
