package models

import (
	"fmt"
	"strings"
)

type FieldExpression interface {
	fmt.Stringer

	JSONPathExpression() string
	IsCondition() bool
}

type FieldAttrValue struct {
	field string
	value any
}

var _ FieldExpression = (*FieldAttrValue)(nil)

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

type FieldArrIndex struct {
	index int
}

var _ FieldExpression = (*FieldArrIndex)(nil)

func (f FieldArrIndex) IsCondition() bool {
	return false
}

func (f FieldArrIndex) String() string {
	return fmt.Sprintf("%d", f.index)
}

func (f FieldArrIndex) JSONPathExpression() string {
	return fmt.Sprintf("%d", f.index)
}

func NewFieldArrIndex(index int) FieldArrIndex {
	return FieldArrIndex{index: index}
}

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
