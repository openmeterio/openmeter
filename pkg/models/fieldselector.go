package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FieldExpression interface {
	fmt.Stringer

	JSONPathExpression() string
	IsCondition() bool
}

var _ FieldExpression = (*FieldAttrValue)(nil)

type FieldAttrValue struct {
	field string
	value any
}

func (f FieldAttrValue) IsCondition() bool {
	return true
}

func (f FieldAttrValue) valueString() string {
	b := strings.Builder{}

	switch v := f.value.(type) {
	case int, int32, int64, float32, float64:
		b.WriteString(fmt.Sprintf("%v", v))
	case string:
		b.WriteString(v)
	case fmt.Stringer:
		b.WriteString(v.String())
	default:
		b.WriteString("!UNSUPPORTED")
	}

	return b.String()
}

func (f FieldAttrValue) String() string {
	return f.field + "=" + f.valueString()
}

func (f FieldAttrValue) JSONPathExpression() string {
	b := strings.Builder{}
	b.WriteString("@.")
	b.WriteString(f.field)
	b.WriteString("=='")
	b.WriteString(f.valueString())
	b.WriteString("'")

	return b.String()
}

func NewFieldAttrValue(field string, value any) FieldAttrValue {
	return FieldAttrValue{field: field, value: value}
}

var _ FieldExpression = (*FieldAttrValue)(nil)

type MultiFieldAttrValue struct {
	values []FieldAttrValue
}

func (m MultiFieldAttrValue) IsCondition() bool {
	return true
}

func (m MultiFieldAttrValue) String() string {
	b := strings.Builder{}

	for idx, attr := range m.values {
		if idx > 0 {
			b.WriteString(", ")
		}

		b.WriteString(attr.String())
	}

	return b.String()
}

func (m MultiFieldAttrValue) JSONPathExpression() string {
	b := strings.Builder{}

	if len(m.values) > 0 {
		for idx, attr := range m.values {
			if idx > 0 {
				b.WriteString(" && ")
			}

			b.WriteString(attr.JSONPathExpression())
		}
	}

	return b.String()
}

func NewMultiFieldAttrValue(values ...FieldAttrValue) MultiFieldAttrValue {
	return MultiFieldAttrValue{values: values}
}

var _ FieldExpression = (*wildCard)(nil)

var WildCard = wildCard{}

type wildCard struct{}

func (w wildCard) IsCondition() bool {
	return false
}

func (w wildCard) String() string {
	return ""
}

func (w wildCard) JSONPathExpression() string {
	return "*"
}

var _ fmt.Stringer = (*FieldSelector)(nil)

type FieldSelector struct {
	field string
	exp   FieldExpression
}

func (p FieldSelector) WithExpression(exp FieldExpression) FieldSelector {
	return FieldSelector{
		field: p.field,
		exp:   exp,
	}
}

func (p FieldSelector) String() string {
	b := strings.Builder{}
	b.WriteString(p.field)

	if p.exp != nil {
		if exp := p.exp.String(); exp != "" {
			b.WriteString("[")
			b.WriteString(exp)
			b.WriteString("]")
		}
	}

	return b.String()
}

func (p FieldSelector) JSONPath() string {
	b := strings.Builder{}
	b.WriteString(p.field)

	if p.exp != nil {
		expOpen := "["
		expClose := "]"

		if p.exp.IsCondition() {
			expOpen = "[?("
			expClose = ")]"
		}

		if exp := p.exp.JSONPathExpression(); exp != "" {
			b.WriteString(expOpen)
			b.WriteString(exp)
			b.WriteString(expClose)
		}
	}

	return b.String()
}

func NewFieldSelector(field string) FieldSelector {
	return FieldSelector{field: field}
}

var (
	_ fmt.Stringer   = (*FieldSelectors)(nil)
	_ json.Marshaler = (*FieldSelectors)(nil)
)

type FieldSelectors []FieldSelector

func (s FieldSelectors) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.JSONPath())
}

func (s FieldSelectors) JSONPath() string {
	if len(s) == 0 {
		return ""
	}

	b := strings.Builder{}
	b.WriteString("$")

	for _, selector := range s {
		b.WriteString(".")
		b.WriteString(selector.JSONPath())
	}

	return b.String()
}

func (s FieldSelectors) String() string {
	if len(s) == 0 {
		return ""
	}

	b := strings.Builder{}
	for idx, selector := range s {
		if idx > 0 {
			b.WriteString(".")
		}

		b.WriteString(selector.String())
	}

	return b.String()
}

func (s FieldSelectors) WithPrefix(prefix FieldSelectors) FieldSelectors {
	return append(prefix, s...)
}

func NewFieldSelectors(selectors ...FieldSelector) FieldSelectors {
	return selectors
}
